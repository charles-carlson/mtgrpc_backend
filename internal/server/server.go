package server

import (
	"context"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/store"
	"backend_nonsense/pb"
)

type Server struct {
	pb.UnimplementedMTGRPCServer
	cards *cards.Service
	store *store.Store
}

func New(cards *cards.Service, store *store.Store) *Server {
	return &Server{cards: cards, store: store}
}

func (s *Server) AddCard(ctx context.Context, req *pb.AddCardRequest) (*pb.AddCardResponse, error) {
	return nil, nil
}

func (s *Server) GetCard(ctx context.Context, req *pb.GetCardRequest) (*pb.GetCardResponse, error) {
	return nil, nil
}

func (s *Server) GetCardsByName(ctx context.Context, req *pb.GetCardsByNameRequest) (*pb.GetCardsByNameResponse, error) {
	return nil, nil
}

func (s *Server) GetCardsBySet(ctx context.Context, req *pb.GetCardsBySetRequest) (*pb.GetCardsBySetResponse, error) {
	return nil, nil
}

func (s *Server) SearchCards(ctx context.Context, req *pb.SearchCardsRequest) (*pb.SearchCardsResponse, error) {
	return nil, nil
}

func (s *Server) ListCards(ctx context.Context, req *pb.ListCardsRequest) (*pb.ListCardsResponse, error) {
	return nil, nil
}
