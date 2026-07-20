package server

import (
	"context"
	"log"

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
func toProtoSetCompletion(s cards.SetCompletion) *pb.SetCompletion {
	return &pb.SetCompletion{
		Set:   s.Set,
		Owned: int32(s.Owned), // proto ints are int32; yours are int
		Total: int32(s.Total),
	}
}

func toProtoSetCompletions(cs []cards.SetCompletion) []*pb.SetCompletion {
	out := make([]*pb.SetCompletion, len(cs))
	for i, c := range cs {
		out[i] = toProtoSetCompletion(c)
	}
	return out
}

type cardService interface {
	GetCard(ctx context.Context, name, set, number string) (*store.Card, error)
	SearchCards(ctx context.Context, name, set string, colors []string, rarity []string, pageSize int32, pageToken string) ([]store.Card, string, error)
	ListCards(ctx context.Context, pageSize int32, pageToken string) ([]store.Card, string, error)
	ListSets(ctx context.Context) ([]string, error)
	GetSetInfo(ctx context.Context) ([]cards.SetCompletion, error)
}

type Server struct {
	pb.UnimplementedMTGRPCServer
	cards cardService
}

var (
	errGetCard            = status.Errorf(codes.Internal, "Unable to retrieve the card")
	errQueryCardsInvalid  = status.Errorf(codes.InvalidArgument, "Invalid arguments given to retrieve cards")
	errQueryCardsInternal = status.Errorf(codes.Internal, "Unable to query cards")
	errListCards          = status.Errorf(codes.Internal, "Unable to fetch collection")
	errListSets           = status.Errorf(codes.Internal, "Unable to retrieve set information")
	errGetSetInfo         = status.Errorf(codes.Internal, "unable to retrieve set completion data")
)

func New(svc cardService) *Server {
	return &Server{cards: svc}
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

func (s *Server) GetSetInfo(ctx context.Context, req *pb.GetSetInfoRequest) (*pb.GetSetInfoResponse, error) {
	results, err := s.cards.GetSetInfo(ctx)
	if err != nil {
		return nil, errGetSetInfo
	}
	return &pb.GetSetInfoResponse{Sets: toProtoSetCompletions(results)}, nil
}
