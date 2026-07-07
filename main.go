package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/eject"
	"backend_nonsense/internal/ingest"
	"backend_nonsense/internal/scryfall"
	"backend_nonsense/internal/server"
	"backend_nonsense/internal/store"
	"backend_nonsense/pb"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const addr = ":50051"

func main() {
	ingestPath := flag.String("ingest", "", "path to Manabox JSON export to ingest (optional)")
	ejectPath := flag.String("eject", "", "path to file to eject cards from store (optional)")
	refresh := flag.Bool("refresh", false, "refresh prices for all cards from Scryfall")
	local := flag.Bool("local", false, "use local DynamoDB at localhost:8000")

	flag.Parse()
	var dbOpts []func(o *dynamodb.Options)
	if *local {
		dbOpts = append(dbOpts, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String("http://localhost:8000")
		})
	}

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}

	db := dynamodb.NewFromConfig(cfg, dbOpts...)
	s := store.New(db)
	sc := scryfall.New()
	cardSvc := cards.NewService(s, sc)

	if *ingestPath != "" {
		log.Printf("ingesting from %s", *ingestPath)
		if err := ingest.RunFile(ctx, *ingestPath, cardSvc); err != nil {
			log.Fatalf("ingest: %v", err)
		}
		log.Println("ingest complete")
	}
	if *ejectPath != "" {
		log.Printf("ejecting from %s", *ejectPath)
		if err := eject.RunFile(ctx, *ejectPath, cardSvc); err != nil {
			log.Fatalf("eject: %v", err)
		}
		log.Println("ejection completed")
	}
	if *refresh {
		log.Println("refreshing prices...")
		if err := cardSvc.RefreshPrices(ctx); err != nil {
			log.Fatalf("refresh: %v", err)
		}
		log.Println("prices refreshed")
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)
	logger := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Printf("method=%s duration=%s code=%s", info.FullMethod, time.Since(start), status.Code(err))
		return resp, err
	}
	interceptor := func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !limiter.Allow() {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded on service")
		}
		return handler(ctx, req)
	}
	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(logger, interceptor))
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, healthSrv)
	healthSrv.SetServingStatus("cards.MTGRPC", grpc_health_v1.HealthCheckResponse_SERVING)
	pb.RegisterMTGRPCServer(srv, server.New(cardSvc))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Printf("gRPC server listening on %s", addr)
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")
	healthSrv.SetServingStatus("cards.MTGRPC", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	srv.GracefulStop()
	log.Println("stopped")
}
