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

func toSortBy(s pb.SortBy) cards.SortBy {
	switch s {
	case pb.SortBy_SORT_UNSPECIFIED:
		return cards.SORT_UNSPECIFIED
	case pb.SortBy_NAME_ASC:
		return cards.NAME_ASC
	case pb.SortBy_NAME_DESC:
		return cards.NAME_DESC
	case pb.SortBy_PRICE_ASC:
		return cards.PRICE_ASC
	case pb.SortBy_PRICE_DESC:
		return cards.PRICE_DESC
	default:
		return cards.NAME_ASC
	}
}
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
		ImageUri: s.ImageURI,
		Set:      s.Set,
		Owned:    int32(s.Owned), // proto ints are int32; yours are int
		Total:    int32(s.Total),
	}
}

func toProtoSetCompletions(cs []cards.SetCompletion) []*pb.SetCompletion {
	out := make([]*pb.SetCompletion, len(cs))
	for i, c := range cs {
		out[i] = toProtoSetCompletion(c)
	}
	return out
}
func toProtoStatInfo(stats cards.CollectionStats) *pb.GetStatInfoResponse {
	typeDist := make(map[string]int32)
	for k, v := range stats.TypeDist {
		typeDist[k] = int32(v)
	}

	colorDist := &pb.ColorDistribution{
		White:     int32(stats.ColorDist.White),
		Blue:      int32(stats.ColorDist.Blue),
		Red:       int32(stats.ColorDist.Red),
		Green:     int32(stats.ColorDist.Green),
		Black:     int32(stats.ColorDist.Black),
		Colorless: int32(stats.ColorDist.Colorless),
	}

	rarityDist := &pb.RarityDistribution{
		Common:   int32(stats.RarityDist.Common),
		Uncommon: int32(stats.RarityDist.Uncommon),
		Rare:     int32(stats.RarityDist.Rare),
		Mythic:   int32(stats.RarityDist.Mythic),
		Special:  int32(stats.RarityDist.Special),
		Bonus:    int32(stats.RarityDist.Bonus),
	}
	subTypeDist := make(map[string]*pb.SubTypeDistribution)
	for cardType, subTypes := range stats.SubTypeDist {
		subTypeDist[cardType] = &pb.SubTypeDistribution{
			Counts: make(map[string]int32),
		}
		for subtype, count := range subTypes {
			subTypeDist[cardType].Counts[subtype] = int32(count)
		}
	}

	return &pb.GetStatInfoResponse{TotalNetWorth: stats.TotalNetWorth, TopKCards: toProtoCards(stats.TopKCards),
		RarityDist: rarityDist, ColorDist: colorDist, TypeDist: typeDist, SubtypeDist: subTypeDist,
	}
}

type cardService interface {
	GetCard(ctx context.Context, name, set, number string) (*store.Card, error)
	SearchCards(ctx context.Context, name string, sets []string, colors []string, rarity []string, pageSize int32, pageToken string, sortBy cards.SortBy) ([]store.Card, string, error)
	ListCards(ctx context.Context, pageSize int32, pageToken string) ([]store.Card, string, error)
	ListSets(ctx context.Context) ([]string, error)
	GetSetInfo(ctx context.Context) ([]cards.SetCompletion, error)
	GetStatInfo(context.Context) (*cards.CollectionStats, error)
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
	errGetSetInfo         = status.Errorf(codes.Internal, "Unable to retrieve set completion data")
	errGetStatInfo        = status.Errorf(codes.Internal, "Unable to retrieve stat info data")
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
	results, nextToken, err := s.cards.SearchCards(ctx, req.Name, req.Set, req.Colors, req.Rarity, req.PageSize, req.PageToken, toSortBy(req.SortBy))
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
func (s *Server) GetStatInfo(ctx context.Context, req *pb.GetStatInfoRequest) (*pb.GetStatInfoResponse, error) {
	results, err := s.cards.GetStatInfo(ctx)
	if err != nil {
		return nil, errGetStatInfo
	}
	protoResponse := toProtoStatInfo(*results)
	return protoResponse, nil
}
