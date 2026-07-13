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

// Refreshing prices for DynamoDb
func (svc *Service) RefreshPrices(ctx context.Context) error {
	cards, err := svc.store.ScanAllCards(ctx)
	if err != nil {
		return err
	}
	for _, card := range cards {
		info, err := svc.scryfall.GetCardInfo(ctx, card.Set, card.Number)
		if err != nil {
			log.Printf("warn: scryfall data for %q (%s/%s): %v", card.Name, card.Set, card.Number, err)
		} else {
			card.Prices = store.Prices{
				USD:     info.Prices.USD,
				USDFoil: info.Prices.USDFoil,
				EUR:     info.Prices.EUR,
				EURFoil: info.Prices.EURFoil,
				TIX:     info.Prices.TIX,
			}
			err := svc.store.UpdatePrices(ctx, card)
			if err != nil {
				log.Printf("warn: update pricing data for %q (%s/%s): %v", card.Name, card.Set, card.Number, err)
			}
		}
	}
	return nil
}

// AddCard fetches Scryfall data (image URL + prices) and writes the card to DynamoDB.
func (svc *Service) AddCard(ctx context.Context, card store.Card) error {
	info, err := svc.scryfall.GetCardInfo(ctx, card.Set, card.Number)
	if err != nil {
		log.Printf("warn: scryfall data for %q (%s/%s): %v", card.Name, card.Set, card.Number, err)
	} else {
		card.ImageURL = info.ImageURL
		card.Colors = info.Colors
		card.Rarity = info.Rarity
		card.Prices = store.Prices{
			USD:     info.Prices.USD,
			USDFoil: info.Prices.USDFoil,
			EUR:     info.Prices.EUR,
			EURFoil: info.Prices.EURFoil,
			TIX:     info.Prices.TIX,
		}
	}

	return svc.store.PutCard(ctx, card)
}
func (svc *Service) RemoveCard(ctx context.Context, card store.Card) error {
	return svc.store.RemoveCard(ctx, card)
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
func (svc *Service) GetCardsBySet(ctx context.Context, set string, pageSize int32, pageToken string) ([]store.Card, string, error) {
	return svc.store.QueryBySet(ctx, set, pageSize, pageToken)
}

// ListCards returns all cards in the collection.
func (svc *Service) ListCards(ctx context.Context, pageSize int32, pageToken string) ([]store.Card, string, error) {
	return svc.store.ScanAll(ctx, pageSize, pageToken)
}

// SearchCards queries the collection with optional name, set, and color filters.
func (svc *Service) SearchCards(ctx context.Context, name, set string, colors []string, rarity []string, pageSize int32, pageToken string) ([]store.Card, string, error) {
	return svc.store.Search(ctx, store.SearchFilter{
		Name:   name,
		Set:    set,
		Colors: colors,
		Rarity: rarity,
	}, pageSize, pageToken)
}
