package server

import (
	"context"
	"errors"
	"testing"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/store"
	"backend_nonsense/pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// stubCardService implements cardService for testing.
type stubCardService struct {
	getCard        *store.Card
	getCardsByName []store.Card
	getCardsBySet  []store.Card
	searchCards    []store.Card
	listCards      []store.Card
	listSets       []string
	getSetInfo     []cards.SetCompletion
	getStatInfo    *cards.CollectionStats
	getErr         error
	searchErr      error
	listErr        error
	listSetsErr    error
	getSetInfoErr  error
	getStatInfoErr error
}

func (s *stubCardService) GetCard(_ context.Context, _, _, _ string) (*store.Card, error) {
	return s.getCard, s.getErr
}
func (s *stubCardService) SearchCards(_ context.Context, _ string, _ []string, _ []string, _ []string, _ int32, _ string) ([]store.Card, string, error) {
	return s.searchCards, "", s.searchErr
}
func (s *stubCardService) ListCards(_ context.Context, _ int32, _ string) ([]store.Card, string, error) {
	return s.listCards, "", s.listErr
}
func (s *stubCardService) ListSets(_ context.Context) ([]string, error) {
	return s.listSets, s.listSetsErr
}
func (s *stubCardService) GetSetInfo(_ context.Context) ([]cards.SetCompletion, error) {
	return s.getSetInfo, s.getSetInfoErr
}
func (s *stubCardService) GetStatInfo(_ context.Context) (*cards.CollectionStats, error) {
	return s.getStatInfo, s.getStatInfoErr
}
func TestSearchCards_Success(t *testing.T) {
	srv := New(&stubCardService{
		searchCards: []store.Card{
			{Name: "Sol Ring", Set: "C21", Number: "149", Count: 2},
			{Name: "Sol Talisman", Set: "MH2", Number: "236", Count: 1},
		},
	})
	resp, err := srv.SearchCards(context.Background(), &pb.SearchCardsRequest{
		Name: "Sol",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Cards) != 2 {
		t.Errorf("got %d cards, want 2", len(resp.Cards))
	}
	if resp.Cards[0].Set != "C21" {
		t.Errorf("got set %q, want %q", resp.Cards[0].Set, "C21")
	}
	if resp.Cards[1].Set != "MH2" {
		t.Errorf("got set %q, want %q", resp.Cards[1].Set, "MH2")
	}
}

func TestSearchCards_InternalRequest(t *testing.T) {
	srv := New(&stubCardService{searchErr: errors.New("dynamo down")})

	_, err := srv.SearchCards(context.Background(), &pb.SearchCardsRequest{Name: "Sol"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if status.Code(err) != codes.Internal {
		t.Errorf("got code %v, want %v", status.Code(err), codes.Internal)
	}
}
func TestListCards_Success(t *testing.T) {
	srv := New(&stubCardService{
		listCards: []store.Card{
			{Name: "Sol Ring", Set: "C21", Number: "263", Count: 6, Prices: store.Prices{USD: "0.50"}},
			{Name: "Black Lotus", Set: "LEA", Number: "232", Count: 3, Prices: store.Prices{USD: "4500.00"}},
		},
	})

	resp, err := srv.ListCards(context.Background(), &pb.ListCardsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Cards) != 2 {
		t.Errorf("got %d cards, want 2", len(resp.Cards))
	}
	if resp.Cards[0].Name != "Sol Ring" {
		t.Errorf("got name %q, want %q", resp.Cards[0].Name, "Sol Ring")
	}
	if resp.Cards[1].Prices.Usd != "4500.00" {
		t.Errorf("got usd %q, want %q", resp.Cards[1].Prices.Usd, "4500.00")
	}
	if resp.NextPageToken != "" {
		t.Errorf("expected empty next_page_token, got %q", resp.NextPageToken)
	}
}

func TestListCards_InternalRequest(t *testing.T) {
	srv := New(&stubCardService{listErr: errors.New("dynamo down")})

	_, err := srv.ListCards(context.Background(), &pb.ListCardsRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if status.Code(err) != codes.Internal {
		t.Errorf("got code %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestGetCard_Success(t *testing.T) {
	srv := New(&stubCardService{
		getCard: &store.Card{
			Name: "Sol Ring", Set: "C21", Number: "263", Count: 2,
			Prices: store.Prices{USD: "0.50"},
		},
	})

	resp, err := srv.GetCard(context.Background(), &pb.GetCardRequest{
		Name: "Sol Ring", Set: "C21", Number: "263",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Card.Name != "Sol Ring" {
		t.Errorf("got name %q, want %q", resp.Card.Name, "Sol Ring")
	}
	if resp.Card.Count != 2 {
		t.Errorf("got count %d, want 2", resp.Card.Count)
	}
	if resp.Card.Prices.Usd != "0.50" {
		t.Errorf("got usd %q, want %q", resp.Card.Prices.Usd, "0.50")
	}
}

func TestGetCard_InvalidRequest(t *testing.T) {
	srv := New(&stubCardService{})

	_, err := srv.GetCard(context.Background(), &pb.GetCardRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("got code %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestGetCard_InternalRequest(t *testing.T) {
	srv := New(&stubCardService{getErr: errors.New("dynamo down")})

	_, err := srv.GetCard(context.Background(), &pb.GetCardRequest{
		Name: "Sol Ring", Set: "C21", Number: "263",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if status.Code(err) != codes.Internal {
		t.Errorf("got code %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestGetSetInfo_Success(t *testing.T) {
	srv := New(&stubCardService{
		getSetInfo: []cards.SetCompletion{
			{Set: "C21", Owned: 2, Total: 3},
			{Set: "MH2", Owned: 5, Total: 10},
		},
	})

	resp, err := srv.GetSetInfo(context.Background(), &pb.GetSetInfoRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Sets) != 2 {
		t.Errorf("got %d sets, want 2", len(resp.Sets))
	}
	if resp.Sets[0].Set != "C21" {
		t.Errorf("got set %q, want %q", resp.Sets[0].Set, "C21")
	}
	if resp.Sets[0].Owned != 2 {
		t.Errorf("got owned %d, want 2", resp.Sets[0].Owned)
	}
	if resp.Sets[1].Total != 10 {
		t.Errorf("got total %d, want 10", resp.Sets[1].Total)
	}
}

func TestGetSetInfo_InternalRequest(t *testing.T) {
	srv := New(&stubCardService{getSetInfoErr: errors.New("scryfall down")})

	_, err := srv.GetSetInfo(context.Background(), &pb.GetSetInfoRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if status.Code(err) != codes.Internal {
		t.Errorf("got code %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestGetStatInfo_Success(t *testing.T) {
	srv := New(&stubCardService{
		getStatInfo: &cards.CollectionStats{
			TotalNetWorth: 1234.56,
			TopKCards: []store.Card{
				{
					Name:   "Black Lotus",
					Set:    "LEA",
					Number: "233",
					Count:  1,
					Prices: store.Prices{USD: "45000.00"},
				},
				{
					Name:   "Sol Ring",
					Set:    "C21",
					Number: "263",
					Count:  4,
					Prices: store.Prices{USD: "0.50"},
				},
			},
			ColorDist: cards.ColorDistribution{
				White: 10,
				Blue:  5,
				Red:   7,
			},
			RarityDist: cards.RarityDistribution{
				Common:   100,
				Uncommon: 50,
				Rare:     20,
				Mythic:   5,
			},
			TypeDist: map[string]int{
				"Creature": 42,
				"Artifact": 15,
			},
			SubTypeDist: map[string]map[string]int{
				"Creature": {
					"Elf":    8,
					"Goblin": 4,
				},
				"Artifact": {
					"Equipment": 3,
				},
			},
		},
	})

	resp, err := srv.GetStatInfo(context.Background(), &pb.GetStatInfoRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalNetWorth != 1234.56 {
		t.Errorf("got total_net_worth %v, want 1234.56", resp.TotalNetWorth)
	}

	if len(resp.TopKCards) != 2 {
		t.Fatalf("got %d cards, want 2", len(resp.TopKCards))
	}

	if resp.TopKCards[0].Name != "Black Lotus" {
		t.Errorf("got %q, want %q", resp.TopKCards[0].Name, "Black Lotus")
	}

	if resp.ColorDist.White != 10 {
		t.Errorf("got white=%d, want 10", resp.ColorDist.White)
	}

	if resp.RarityDist.Mythic != 5 {
		t.Errorf("got mythic=%d, want 5", resp.RarityDist.Mythic)
	}

	if resp.TypeDist["Creature"] != 42 {
		t.Errorf("got creature=%d, want 42", resp.TypeDist["Creature"])
	}

	if resp.SubtypeDist["Creature"].Counts["Elf"] != 8 {
		t.Errorf("got elf=%d, want 8", resp.SubtypeDist["Creature"].Counts["Elf"])
	}
}
func TestGetStatInfo_InternalRequest(t *testing.T) {
	srv := New(&stubCardService{
		getStatInfoErr: errors.New("stats unavailable"),
	})

	_, err := srv.GetStatInfo(context.Background(), &pb.GetStatInfoRequest{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status.Code(err) != codes.Internal {
		t.Errorf("got code %v, want %v", status.Code(err), codes.Internal)
	}
}
