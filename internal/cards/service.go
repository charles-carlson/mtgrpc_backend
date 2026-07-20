package cards

import (
	"context"
	"encoding/base64"
	"log"
	"slices"
	"sort"
	"strconv"
	"strings"

	"backend_nonsense/internal/scryfall"
	"backend_nonsense/internal/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errLoadingSnapshot = status.Errorf(codes.Internal, "Unable to load current snapshot from cache")

type SetCompletion struct {
	ImageURI string
	Set      string
	Owned    int
	Total    int // printed_size; 0 = unknown (Scryfall data missing), using nil pointer for print_size in snapshot
}

// Service handles card operations shared across ingest and manual entry.
type Service struct {
	store    *store.Store
	scryfall *scryfall.Client
	cache    cardsCache
	setc     setCache
}

func NewService(s *store.Store, sc *scryfall.Client) *Service {
	return &Service{store: s, scryfall: sc}
}
func (svc *Service) Reload(ctx context.Context) error {
	if err := svc.cache.reload(ctx, svc.store.ScanAllCards); err != nil {
		return err
	}
	snap := svc.cache.current()
	log.Printf("snapshot loaded: %d cards, %d sets", len(snap.allCards), len(snap.sets))
	return nil
}

func (svc *Service) ReloadSetInfo(ctx context.Context) error {
	if err := svc.setc.reload(ctx, svc.scryfall.GetSetsInfo); err != nil {
		return err
	}
	setmd := svc.setc.current()
	log.Printf("set metadata loaded: %d sets", len(*setmd))
	return nil
}

func (svc *Service) GetSetInfo(context.Context) ([]SetCompletion, error) {
	snap := svc.cache.current()
	if snap == nil {
		return nil, errLoadingSnapshot
	}
	setmd := svc.setc.current()
	// may be nil, but will let it be non fatal
	// set code -> collector number -> true/false
	owned := make(map[string]map[int]struct{})
	for _, c := range snap.allCards {
		n, err := strconv.Atoi(c.Number)
		if err != nil {
			continue
		}
		if owned[c.Set] == nil {
			owned[c.Set] = make(map[int]struct{})
		}
		owned[c.Set][n] = struct{}{}
	}
	var out []SetCompletion
	//Iterate through each set code in snapshot, sets are already sorted, output result will be an array sorted with set results
	for _, code := range snap.sets {
		//set the total to 0 before counting printed size of the set
		total := 0
		imageURI := ""
		//if scryfall data does not exist silently fail
		if setmd != nil {
			//if not, check if code exists in the map, and set total to Printed Size if exists in set info
			if info, ok := (*setmd)[code]; ok && info.PrintedSize != nil {
				total = *info.PrintedSize
				imageURI = info.IconSVGUri
			}
		}
		have := 0
		//count cards in collection that exist
		for n := range owned[code] {
			if total == 0 || (n >= 1 && n <= total) {
				have++
			}
		}
		//append result
		out = append(out, SetCompletion{ImageURI: imageURI, Set: code, Owned: have, Total: total})
	}
	return out, nil
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
	snap := svc.cache.current()
	if snap == nil {
		return nil, errLoadingSnapshot
	}
	c, ok := snap.byKey[keyOf(store.Card{Name: name, Set: set, Number: number})]
	if !ok {
		return nil, nil // miss: nil card, no error (server maps to an empty response)
	}
	return &c, nil
}

// ListCards returns all cards in the collection.
func (svc *Service) ListCards(ctx context.Context, pageSize int32, pageToken string) ([]store.Card, string, error) {
	snap := svc.cache.current()
	if snap == nil {
		return nil, "", errLoadingSnapshot
	}
	return paginate(snap.allCards, pageSize, pageToken)
}

// SearchCards queries the collection with optional name, set, and color filters.
func (svc *Service) SearchCards(ctx context.Context, name, set string, colors []string, rarity []string, pageSize int32, pageToken string) ([]store.Card, string, error) {
	snap := svc.cache.current()
	if snap == nil {
		return nil, "", errLoadingSnapshot
	}
	filtered := buildFilteredCards(snap.allCards, store.SearchFilter{
		Name:   name,
		Set:    set,
		Colors: colors,
		Rarity: rarity,
	})
	return paginate(filtered, pageSize, pageToken)
}

func (svc *Service) ListSets(ctx context.Context) ([]string, error) {
	snap := svc.cache.current()
	if snap == nil {
		return nil, errLoadingSnapshot
	}
	return snap.sets, nil
}

// First index whose key is strictly greater than the cursor.
// sort.Search needs a monotonic predicate — true for all keys past the cursor.
func paginate(sorted []store.Card, pageSize int32, token string) ([]store.Card, string, error) {
	cursor, err := decodeCursor(token)
	if err != nil {
		return nil, "", err
	}
	start := sort.Search(len(sorted), func(i int) bool {
		return keyOf(sorted[i]) > cursor
	})
	end := len(sorted)
	if pageSize > 0 && start+int(pageSize) < end {
		end = start + int(pageSize)
	}
	page := sorted[start:end:end] //3 index cap
	next := ""
	if end < len(sorted) {
		//next page key is at the last value to be returned in the page
		next = encodeCursor(keyOf(sorted[end-1]))
	}
	return page, next, nil
}
func encodeCursor(key string) string { return base64.StdEncoding.EncodeToString([]byte(key)) }

func decodeCursor(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func buildFilteredCards(cards []store.Card, filter store.SearchFilter) []store.Card {
	filtered := make([]store.Card, 0, len(cards))
	for _, card := range cards {
		if matches(card, filter) {
			filtered = append(filtered, card)
		}
	}
	return filtered
}
func matches(c store.Card, f store.SearchFilter) bool {
	if f.Name != "" && !strings.Contains(c.Name, f.Name) {
		return false // fails the name group
	}
	if f.Set != "" && c.Set != f.Set {
		return false // fails the set group
	}
	if len(f.Rarity) > 0 && !slices.Contains(f.Rarity, c.Rarity) {
		return false // fails the rarity group (OR handled by Contains)
	}
	if len(f.Colors) > 0 && !hasAnyColor(c.Colors, f.Colors) {
		return false // fails the color group
	}
	return true // passed every active group
}

func hasAnyColor(cardColors, want []string) bool {
	for _, w := range want {
		if slices.Contains(cardColors, w) {
			return true // OR within the color group
		}
	}
	return false
}
