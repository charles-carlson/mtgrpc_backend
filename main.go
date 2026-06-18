package main

import (
	"context"
	"flag"
	"log"
	"net"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"google.golang.org/grpc"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/ingest"
	"backend_nonsense/internal/scryfall"
	"backend_nonsense/internal/store"
)

const addr = ":50051"

func main() {
	ingestPath := flag.String("ingest", "", "path to Manabox JSON export to ingest (optional)")
	flag.Parse()

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}

	db := dynamodb.NewFromConfig(cfg)
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

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()

	// TODO: register your services here, passing cardSvc for manual entry
	// pb.RegisterYourServiceServer(srv, &yourServiceImpl{cards: cardSvc, store: s})

	log.Printf("gRPC server listening on %s", addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
