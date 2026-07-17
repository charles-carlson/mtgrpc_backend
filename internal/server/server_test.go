package server

import (
	"context"
	"errors"
	"testing"

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
	getErr         error
	searchErr      error
	listErr        error
	listSetsErr    error
}

func (s *stubCardService) GetCard(_ context.Context, _, _, _ string) (*store.Card, error) {
	return s.getCard, s.getErr
}
func (s *stubCardService) SearchCards(_ context.Context, _, _ string, _ []string, _ []string, _ int32, _ string) ([]store.Card, string, error) {
	return s.searchCards, "", s.searchErr
}
func (s *stubCardService) ListCards(_ context.Context, _ int32, _ string) ([]store.Card, string, error) {
	return s.listCards, "", s.listErr
}
func (s *stubCardService) ListSets(_ context.Context) ([]string, error) {
	return s.listSets, s.listSetsErr
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