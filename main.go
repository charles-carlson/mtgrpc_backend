package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
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
	"backend_nonsense/pb/pbconnect"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/time/rate"
)

func main() {
	// different --addr when running with --local.
	addr := flag.String("addr", ":7200", "address for the Connect/gRPC/gRPC-Web server")
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
	if err := cardSvc.Reload(ctx); err != nil {
		log.Fatalf("reload: %v", err)
	}
	if err := cardSvc.ReloadSetInfo(ctx); err != nil {
		log.Printf("reload set: %v", err)
	}
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)
	logging := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			code := "ok"
			if err != nil {
				code = connect.CodeOf(err).String()
			}
			log.Printf("method=%s duration=%s code=%s", req.Spec().Procedure, time.Since(start), code)
			return resp, err
		}
	})
	rateLimit := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if !limiter.Allow() {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded on service"))
			}
			return next(ctx, req)
		}
	})

	adapter := server.NewConnectAdapter(server.New(cardSvc))
	mux := http.NewServeMux()
	mux.Handle(pbconnect.NewMTGRPCHandler(adapter, connect.WithInterceptors(logging, rateLimit)))
	mux.Handle(grpchealth.NewHandler(grpchealth.NewStaticChecker("cards.MTGRPC")))

	// h2c serves HTTP/2 without TLS so existing gRPC clients keep working,
	// while browsers reach Connect / gRPC-Web over HTTP/1.1 on the same port.
	httpSrv := &http.Server{
		Addr:    *addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Printf("server listening on %s (Connect + gRPC + gRPC-Web)", *addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	log.Println("stopped")
}
