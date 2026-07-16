package server

import (
	"context"
	"log"

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
		Prices: &pb.Prices{
			Usd:     c.Prices.USD,
			UsdFoil: c.Prices.USDFoil,
			Eur:     c.Prices.EUR,
			EurFoil: c.Prices.EURFoil,
			Tix:     c.Prices.TIX,
		},
		Colors: c.Colors,
		Rarity: c.Rarity,
	}
}

func toProtoCards(cs []store.Card) []*pb.Card {
	out := make([]*pb.Card, len(cs))
	for i, c := range cs {
		out[i] = toProtoCard(c)
	}
	return out
}

type cardService interface {
	AddCard(ctx context.Context, card store.Card) error
	GetCard(ctx context.Context, name, set, number string) (*store.Card, error)
	SearchCards(ctx context.Context, name, set string, colors []string, rarity []string, pageSize int32, pageToken string) ([]store.Card, string, error)
	ListCards(ctx context.Context, pageSize int32, pageToken string) ([]store.Card, string, error)
	ListSets(ctx context.Context) ([]string, error)
}

type Server struct {
	pb.UnimplementedMTGRPCServer
	cards cardService
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
	errAddCardInvalid        = status.Errorf(codes.InvalidArgument, "Invalid Add Card requirements")
	errAddCardInternal       = status.Errorf(codes.Internal, "Unable to add card to store")
	errListSets              = status.Errorf(codes.Internal, "Unable to retrieve set information")
)

func New(svc cardService) *Server {
	return &Server{cards: svc}
}

func (s *Server) AddCard(ctx context.Context, req *pb.AddCardRequest) (*pb.AddCardResponse, error) {
	if req.Name == "" && req.Set == "" && req.Number == "" {
		return nil, errAddCardInvalid
	}
	err := s.cards.AddCard(ctx, store.Card{Name: req.Name, Set: req.Set, Number: req.Number, Count: int(req.Count)})
	if err != nil {
		return nil, errAddCardInternal
	}
	card, err := s.cards.GetCard(ctx, req.Name, req.Set, req.Number)
	if err != nil {
		return nil, errGetCard
	}
	return &pb.AddCardResponse{Card: toProtoCard(*card)}, nil
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

func (s *Server) SearchCards(ctx context.Context, req *pb.SearchCardsRequest) (*pb.SearchCardsResponse, error) {
	results, nextToken, err := s.cards.SearchCards(ctx, req.Name, req.Set, req.Colors, req.Rarity, req.PageSize, req.PageToken)
	if err != nil {
		log.Printf("SearchCards(name=%q set=%q colors=%v rarity=%v): %v", req.Name, req.Set, req.Colors, req.Rarity, err)
		return nil, errQueryCardsInternal
	}

	return &pb.SearchCardsResponse{Cards: toProtoCards(results), NextPageToken: nextToken}, nil
}

func (s *Server) ListCards(ctx context.Context, req *pb.ListCardsRequest) (*pb.ListCardsResponse, error) {
	results, nextToken, err := s.cards.ListCards(ctx, req.PageSize, req.PageToken)
	if err != nil {
		return nil, errListCards
	}
	return &pb.ListCardsResponse{Cards: toProtoCards(results), NextPageToken: nextToken}, nil
}

func (s *Server) ListSets(ctx context.Context, req *pb.ListSetsRequest) (*pb.ListSetsResponse, error) {
	results, err := s.cards.ListSets(ctx)
	if err != nil {
		return nil, errListSets
	}
	return &pb.ListSetsResponse{Sets: results}, nil
}
