package server

import (
	"context"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/store"
	"backend_nonsense/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toProtoCard(c store.Card) *pb.Card {
	return &pb.Card{
		Name:     c.Name,
		Set:      c.Set,
		Number:   c.Number,
		Count:    int32(c.Count),
		ImageUrl: c.ImageURL,
	}
}

func toProtoCards(cs []store.Card) []*pb.Card {
	out := make([]*pb.Card, len(cs))
	for i, c := range cs {
		out[i] = toProtoCard(c)
	}
	return out
}

type Server struct {
	pb.UnimplementedMTGRPCServer
	cards *cards.Service
}

var (
	errGetCard               = status.Errorf(codes.Internal, "Unable to retrieve the card")
	errGetCardsByName        = status.Errorf(codes.Internal, "Unable to retrieve cards by name")
	errGetCardsByNameInvalid = status.Errorf(codes.InvalidArgument, "Invalid name argument")
	errGetCardsBySet         = status.Errorf(codes.Internal, "Unable to retrieve cards by set")
	errGetCardsBySetInvalid  = status.Errorf(codes.InvalidArgument, "Invalid set argument")
	errQueryCardsInvalid     = status.Errorf(codes.InvalidArgument, "Invalid arguments given to retrieve cards")
	errQueryCardsInternal    = status.Errorf(codes.Internal, "Unable to query cards")
	errListCards             = status.Errorf(codes.Internal, "Unable to fetch collection")
)

func New(cards *cards.Service) *Server {
	return &Server{cards: cards}
}

func (s *Server) AddCard(ctx context.Context, req *pb.AddCardRequest) (*pb.AddCardResponse, error) {
	return nil, nil
}

func (s *Server) GetCard(ctx context.Context, req *pb.GetCardRequest) (*pb.GetCardResponse, error) {
	if req.Name == "" && req.Set == "" && req.Number == "" {
		return nil, errQueryCardsInvalid
	}

	card, err := s.cards.GetCard(ctx, req.Name, req.Set, req.Number)
	if err != nil {
		return nil, errGetCard
	}
	if card == nil {
		return &pb.GetCardResponse{}, nil
	}

	return &pb.GetCardResponse{Card: toProtoCard(*card)}, nil
}

func (s *Server) GetCardsByName(ctx context.Context, req *pb.GetCardsByNameRequest) (*pb.GetCardsByNameResponse, error) {
	if req.Name == "" {
		return nil, errGetCardsByNameInvalid
	}

	results, err := s.cards.GetCardsByName(ctx, req.Name)
	if err != nil {
		return nil, errGetCardsByName
	}

	return &pb.GetCardsByNameResponse{Cards: toProtoCards(results)}, nil
}

func (s *Server) GetCardsBySet(ctx context.Context, req *pb.GetCardsBySetRequest) (*pb.GetCardsBySetResponse, error) {
	if req.Set == "" {
		return nil, errGetCardsBySetInvalid
	}

	results, err := s.cards.GetCardsBySet(ctx, req.Set)
	if err != nil {
		return nil, errGetCardsBySet
	}

	return &pb.GetCardsBySetResponse{Cards: toProtoCards(results)}, nil

}

func (s *Server) SearchCards(ctx context.Context, req *pb.SearchCardsRequest) (*pb.SearchCardsResponse, error) {
	results, err := s.cards.SearchCards(ctx, req.Name, req.Set, req.Colors)
	if err != nil {
		return nil, errQueryCardsInternal
	}

	return &pb.SearchCardsResponse{Cards: toProtoCards(results)}, nil
}

func (s *Server) ListCards(ctx context.Context, req *pb.ListCardsRequest) (*pb.ListCardsResponse, error) {
	results, err := s.cards.ListCards(ctx)
	if err != nil {
		return nil, errListCards
	}
	return &pb.ListCardsResponse{Cards: toProtoCards(results)}, nil
}
