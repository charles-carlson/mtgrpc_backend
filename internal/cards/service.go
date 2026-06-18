package cards

import (
	"context"
	"log"

	"backend_nonsense/internal/scryfall"
	"backend_nonsense/internal/store"
)

// Service handles card operations shared across ingest and manual entry.
type Service struct {
	store    *store.Store
	scryfall *scryfall.Client
}

func NewService(s *store.Store, sc *scryfall.Client) *Service {
	return &Service{store: s, scryfall: sc}
}

// AddCard fetches the Scryfall image URL and writes the card to DynamoDB.
func (svc *Service) AddCard(ctx context.Context, card store.Card) error {
	imageURL, err := svc.scryfall.GetImageURL(ctx, card.Set, card.Number)
	if err != nil {
		log.Printf("warn: scryfall image for %q (%s/%s): %v", card.Name, card.Set, card.Number, err)
	} else {
		card.ImageURL = imageURL
	}

	return svc.store.PutCard(ctx, card)
}

// GetCard returns a specific card by name, set, and number.
func (svc *Service) GetCard(ctx context.Context, name, set, number string) (*store.Card, error) {
	return svc.store.GetCard(ctx, name, set, number)
}

// GetCardsByName returns all printings of a card across sets.
func (svc *Service) GetCardsByName(ctx context.Context, name string) ([]store.Card, error) {
	return svc.store.QueryByName(ctx, name)
}

// GetCardsBySet returns all cards in a given set.
func (svc *Service) GetCardsBySet(ctx context.Context, set string) ([]store.Card, error) {
	return svc.store.QueryBySet(ctx, set)
}

// ListCards returns all cards in the collection.
func (svc *Service) ListCards(ctx context.Context) ([]store.Card, error) {
	return svc.store.ScanAll(ctx)
}
