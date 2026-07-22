package cards

import (
	"cmp"
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

type ColorDistribution struct {
	White     int
	Blue      int
	Red       int
	Green     int
	Black     int
	Colorless int
}

type RarityDistribution struct {
	Common   int
	Uncommon int
	Rare     int
	Mythic   int
	Special  int
	Bonus    int
}

type CollectionStats struct {
	TotalNetWorth float64
	TopKCards     []store.Card
	RarityDist    RarityDistribution
	ColorDist     ColorDistribution
	TypeDist      map[string]int
	SubTypeDist   map[string]map[string]int
}
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
func CheckPrice(finish string, prices store.Prices) string {
	if finish == "foil" {
		return prices.USDFoil
	} else if finish == "etched" {
		return prices.USDEtched
	} else {
		return prices.USD
	}
}

// WIP, need to set up finish type to ingestion pipeline
func (svc *Service) GetStats(context.Context) (*CollectionStats, error) {
	snap := svc.cache.current()
	if snap == nil {
		return nil, errLoadingSnapshot
	}
	collection := make([]store.Card, len(snap.allCards))
	copy(collection, snap.allCards)
	//Sort cards in decreasing order of cards worth the most value
	// Can grab the first five cards to show most expensive
	slices.SortFunc(collection, func(c1, c2 store.Card) int {
		p1, _ := strconv.ParseFloat(CheckPrice(c1.Finish, c1.Prices), 64)
		p2, _ := strconv.ParseFloat(CheckPrice(c2.Finish, c2.Prices), 64)
		return cmp.Compare(p2, p1)
	})
	totalLiquid := float64(0)
	var rareDist RarityDistribution
	var colorDist ColorDistribution
	typeDist := make(map[string]int)
	subTypeDist := make(map[string]map[string]int)
	for _, card := range collection {
		//Retrieve correct price of card based on finish
		value := CheckPrice(card.Finish, card.Prices)
		price, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		totalLiquid += price
		//Rarity Dis
		switch card.Rarity {
		case "common":
			rareDist.Common++
		case "uncommon":
			rareDist.Uncommon++
		case "rare":
			rareDist.Rare++
		case "mythic":
			rareDist.Mythic++
		case "special":
			rareDist.Special++
		case "bonus":
			rareDist.Bonus++
		}
		n := card.Count
		if len(card.Colors) == 0 {
			colorDist.Colorless += n // empty array = colorless (lands, most artifacts)
		} else {
			for _, col := range card.Colors {
				switch col {
				case "W":
					colorDist.White += n
				case "U":
					colorDist.Blue += n
				case "B":
					colorDist.Black += n
				case "R":
					colorDist.Red += n
				case "G":
					colorDist.Green += n
				}
			}
		}

		//type_line looks like type type type -- subtype subtype subtype...
		// multiple types if em dash exists
		parts := strings.SplitN(card.TypeLine, "\u2014", 2)
		cardTypes := strings.Fields(parts[0])
		var subTypes []string
		if len(parts) == 2 {
			subTypes = strings.Fields(parts[1])
		}
		for _, cType := range cardTypes {
			typeDist[cType]++
			if len(subTypes) > 0 {
				for _, sType := range subTypes {
					if subTypeDist[cType] == nil {
						subTypeDist[cType] = make(map[string]int)
					}
					subTypeDist[cType][sType]++
				}
			}
		}
	}
	k := min(5, len(collection))
	return &CollectionStats{
		ColorDist:     colorDist,
		RarityDist:    rareDist,
		TypeDist:      typeDist,
		SubTypeDist:   subTypeDist,
		TotalNetWorth: totalLiquid,
		TopKCards:     collection[:k],
	}, nil
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
	var completableSetTypes = map[string]struct{}{
		"expansion": {},
		"core":      {},
	}
	//Iterate through each set code in snapshot, sets are already sorted, output result will be an array sorted with set results
	for _, code := range snap.sets {
		if setmd == nil {
			continue
		}
		info, ok := (*setmd)[strings.ToLower(code)] // ToLower = your case fix
		if !ok {
			continue // set not in Scryfall's data → skip
		}
		if _, completable := completableSetTypes[info.SetType]; !completable {
			continue // ← the gate: drop commander/masters/etc.
		}
		//set the total to 0 before counting printed size of the set
		total := 0
		//if scryfall data does not exist silently fail
		//icon is available whenever the set exists in metadata; total only when printed_size is set
		if info.PrintedSize != nil {
			total = *info.PrintedSize
		} else {
			total = info.CardCount
		}
		have := 0
		//count cards in collection that exist
		for n := range owned[code] {
			if total == 0 || (n >= 1 && n <= total) {
				have++
			}
		}
		//append result
		out = append(out, SetCompletion{ImageURI: info.IconSVGUri, Set: code, Owned: have, Total: total})
		log.Printf("Set Completion for %s: owned is %d and total is %d", code, have, total)
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
				USD:       info.Prices.USD,
				USDFoil:   info.Prices.USDFoil,
				EUR:       info.Prices.EUR,
				EURFoil:   info.Prices.EURFoil,
				TIX:       info.Prices.TIX,
				USDEtched: info.Prices.USDEtched,
				EUREtched: info.Prices.EUREtched,
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
		if card.Finish == "" {
			card.Finish = "nonfoil"
		}
		card.ImageURL = info.ImageURL
		card.Colors = info.Colors
		card.Rarity = info.Rarity
		card.TypeLine = info.TypeLine
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
