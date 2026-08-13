package cards

import (
	"backend_nonsense/internal/scryfall"
	"backend_nonsense/internal/store"
	"context"
	"strconv"
	"testing"
)

func newTestService(t *testing.T, cards []store.Card) *Service {
	t.Helper()
	svc := &Service{}
	if err := svc.cache.reload(context.Background(), func(ctx context.Context) ([]store.Card, error) {
		return cards, nil
	}); err != nil {
		t.Fatalf("prime cache: %v", err)
	}
	return svc
}

func TestGetStatInfoSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Lightning Bolt", Set: "M10", Number: "149", Count: 4, ImageURL: "None", Prices: store.Prices{
			USD: "0.15", USDFoil: "0.50", USDEtched: "1.50", EUR: "0.25", EURFoil: "1.00", EUREtched: "3.00",
		}, Colors: []string{"R"}, Rarity: "common", Finish: "nonfoil", TypeLine: "Instant"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if want := 0.60; stats.TotalNetWorth != want {
		t.Errorf("TotalNetWorth = %v, want %v", stats.TotalNetWorth, want)
	}
	if stats.RarityDist.Common != 1 {
		t.Errorf("RarityDist.Common = %d, want 1", stats.RarityDist.Common)
	}
	if stats.ColorDist.Red != 4 {
		t.Errorf("ColorDist.Red = %d, want 4", stats.ColorDist.Red)
	}
	if got := stats.TypeDist["Instant"]; got != 1 {
		t.Errorf("TypeDist[Instant] = %d, want 1", got)
	}
	if len(stats.TopKCards) != 1 {
		t.Errorf("len(TopKCards) = %d, want 1", len(stats.TopKCards))
	}
}

func TestGetStatInfoErrLoadNotReadyError(t *testing.T) {
	svc := &Service{}
	_, err := svc.GetStatInfo(context.Background())
	if err == nil {
		t.Fatal("expected error when snapshot isn't loaded yet")
	}
}

func TestGetStatInfoEmptyCollectionSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if stats.TotalNetWorth != 0 {
		t.Errorf("TotalNetWorth = %v, want 0", stats.TotalNetWorth)
	}
	if len(stats.TopKCards) != 0 {
		t.Errorf("len(TopKCards) = %d, want 0", len(stats.TopKCards))
	}
	if len(stats.TypeDist) != 0 {
		t.Errorf("len(TypeDist) = %d, want 0", len(stats.TypeDist))
	}
}

func TestGetStatInfoCardsWithNoPricesSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Mystery Booster Card", Set: "MB1", Number: "1", Count: 2,
			Prices: store.Prices{USD: ""}, Rarity: "common", Finish: "nonfoil", TypeLine: "Sorcery"},
		{Name: "Weird Promo", Set: "PROMO", Number: "1", Count: 1,
			Prices: store.Prices{USD: "not-a-number"}, Rarity: "rare", Finish: "nonfoil", TypeLine: "Sorcery"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if stats.TotalNetWorth != 0 {
		t.Errorf("TotalNetWorth = %v, want 0 (empty/malformed prices should contribute nothing, not error)", stats.TotalNetWorth)
	}
}

func TestGetStatInfoUnrecognizedRaritySuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Weird Card", Set: "XXX", Number: "1", Count: 1, Rarity: "timeshifted", TypeLine: "Instant"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	want := RarityDistribution{}
	if stats.RarityDist != want {
		t.Errorf("RarityDist = %+v, want all zero (unrecognized rarity shouldn't increment anything)", stats.RarityDist)
	}
}

func TestGetStatInfoAllRaritiesSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "C", Set: "SET", Number: "1", Count: 1, Rarity: "common", TypeLine: "Instant"},
		{Name: "U", Set: "SET", Number: "2", Count: 1, Rarity: "uncommon", TypeLine: "Instant"},
		{Name: "R", Set: "SET", Number: "3", Count: 1, Rarity: "rare", TypeLine: "Instant"},
		{Name: "M", Set: "SET", Number: "4", Count: 1, Rarity: "mythic", TypeLine: "Instant"},
		{Name: "S", Set: "SET", Number: "5", Count: 1, Rarity: "special", TypeLine: "Instant"},
		{Name: "B", Set: "SET", Number: "6", Count: 1, Rarity: "bonus", TypeLine: "Instant"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	want := RarityDistribution{Common: 1, Uncommon: 1, Rare: 1, Mythic: 1, Special: 1, Bonus: 1}
	if stats.RarityDist != want {
		t.Errorf("RarityDist = %+v, want %+v", stats.RarityDist, want)
	}
}

func TestGetStatInfoColorlessSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Wastes", Set: "OGW", Number: "183", Count: 2, Colors: []string{}, Rarity: "common", TypeLine: "Basic Land"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	want := ColorDistribution{Colorless: 2}
	if stats.ColorDist != want {
		t.Errorf("ColorDist = %+v, want %+v", stats.ColorDist, want)
	}
}

func TestGetStatInfoMultiColorCardsSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Azorius Signet", Set: "RAV", Number: "264", Count: 3, Colors: []string{"W", "U"}, Rarity: "common", TypeLine: "Artifact"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	want := ColorDistribution{White: 3, Blue: 3}
	if stats.ColorDist != want {
		t.Errorf("ColorDist = %+v, want %+v — a multicolor card should add its full Count to EVERY one of its colors", stats.ColorDist, want)
	}
}

func TestGetStatInfoSingleTypeSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Lightning Bolt", Set: "M10", Number: "149", Count: 1, Rarity: "common", TypeLine: "Instant"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if got := stats.TypeDist["Instant"]; got != 1 {
		t.Errorf("TypeDist[Instant] = %d, want 1", got)
	}
	if len(stats.SubTypeDist) != 0 {
		t.Errorf("SubTypeDist = %+v, want empty (no em-dash in type line)", stats.SubTypeDist)
	}
}

func TestGetStatInfoMultiTypesSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Llanowar Elder", Set: "TST", Number: "1", Count: 1, Rarity: "rare", TypeLine: "Legendary Creature — Elf Druid"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	wantTypes := map[string]int{"Legendary": 1, "Creature": 1}
	for typ, want := range wantTypes {
		if got := stats.TypeDist[typ]; got != want {
			t.Errorf("TypeDist[%s] = %d, want %d", typ, got, want)
		}
	}
	for _, typ := range []string{"Legendary", "Creature"} {
		for _, sub := range []string{"Elf", "Druid"} {
			if got := stats.SubTypeDist[typ][sub]; got != 1 {
				t.Errorf("SubTypeDist[%s][%s] = %d, want 1", typ, sub, got)
			}
		}
	}
}

func TestGetStatInfoCollectionLessThan5Success(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "A", Set: "SET", Number: "1", Count: 1, Prices: store.Prices{USD: "1.00"}, TypeLine: "Instant"},
		{Name: "B", Set: "SET", Number: "2", Count: 1, Prices: store.Prices{USD: "2.00"}, TypeLine: "Instant"},
		{Name: "C", Set: "SET", Number: "3", Count: 1, Prices: store.Prices{USD: "3.00"}, TypeLine: "Instant"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if len(stats.TopKCards) != 3 {
		t.Errorf("len(TopKCards) = %d, want 3 (a collection smaller than 5 shouldn't be padded or truncated)", len(stats.TopKCards))
	}
}

func TestGetStatInfoCollectonMoreThan5Success(t *testing.T) {
	cards := make([]store.Card, 0, 7)
	for i := 1; i <= 7; i++ {
		cards = append(cards, store.Card{
			Name: "Card", Set: "SET", Number: strconv.Itoa(i), Count: 1,
			Prices: store.Prices{USD: strconv.Itoa(i) + ".00"}, TypeLine: "Instant",
		})
	}
	svc := newTestService(t, cards)

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if len(stats.TopKCards) != 5 {
		t.Errorf("len(TopKCards) = %d, want 5 (capped, even with 7 cards in the collection)", len(stats.TopKCards))
	}
}

func TestGetStatInfoTopK5CardsSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Cheap", Set: "SET", Number: "1", Count: 1, Prices: store.Prices{USD: "1.00"}, TypeLine: "Instant"},
		{Name: "Priciest", Set: "SET", Number: "2", Count: 1, Prices: store.Prices{USD: "50.00"}, TypeLine: "Instant"},
		{Name: "Mid", Set: "SET", Number: "3", Count: 1, Prices: store.Prices{USD: "10.00"}, TypeLine: "Instant"},
		{Name: "Cheaper", Set: "SET", Number: "4", Count: 1, Prices: store.Prices{USD: "0.50"}, TypeLine: "Instant"},
		{Name: "Midlow", Set: "SET", Number: "5", Count: 1, Prices: store.Prices{USD: "5.00"}, TypeLine: "Instant"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if len(stats.TopKCards) != 5 {
		t.Fatalf("len(TopKCards) = %d, want 5", len(stats.TopKCards))
	}
	if got := stats.TopKCards[0].Name; got != "Priciest" {
		t.Errorf("TopKCards[0] = %q, want %q (most expensive first)", got, "Priciest")
	}
}

func TestGetStatInfoTopKLessThan5CardsSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Priciest", Set: "SET", Number: "1", Count: 1, Prices: store.Prices{USD: "20.00"}, TypeLine: "Instant"},
		{Name: "Cheaper", Set: "SET", Number: "2", Count: 1, Prices: store.Prices{USD: "1.00"}, TypeLine: "Instant"},
	})

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if len(stats.TopKCards) != 2 {
		t.Fatalf("len(TopKCards) = %d, want 2", len(stats.TopKCards))
	}
	if got := stats.TopKCards[0].Name; got != "Priciest" {
		t.Errorf("TopKCards[0] = %q, want %q", got, "Priciest")
	}
}

func TestGetStatInfoTopKMorethan5CardsSuccess(t *testing.T) {
	cards := []store.Card{
		{Name: "Card1", Set: "SET", Number: "1", Count: 1, Prices: store.Prices{USD: "1.00"}, TypeLine: "Instant"},
		{Name: "Card2", Set: "SET", Number: "2", Count: 1, Prices: store.Prices{USD: "2.00"}, TypeLine: "Instant"},
		{Name: "Card3", Set: "SET", Number: "3", Count: 1, Prices: store.Prices{USD: "3.00"}, TypeLine: "Instant"},
		{Name: "Card4", Set: "SET", Number: "4", Count: 1, Prices: store.Prices{USD: "4.00"}, TypeLine: "Instant"},
		{Name: "Card5", Set: "SET", Number: "5", Count: 1, Prices: store.Prices{USD: "5.00"}, TypeLine: "Instant"},
		{Name: "TheBest", Set: "SET", Number: "6", Count: 1, Prices: store.Prices{USD: "99.00"}, TypeLine: "Instant"},
		{Name: "Card7", Set: "SET", Number: "7", Count: 1, Prices: store.Prices{USD: "7.00"}, TypeLine: "Instant"},
		{Name: "Card8", Set: "SET", Number: "8", Count: 1, Prices: store.Prices{USD: "8.00"}, TypeLine: "Instant"},
	}
	svc := newTestService(t, cards)

	stats, err := svc.GetStatInfo(context.Background())
	if err != nil {
		t.Fatalf("GetStatInfo: %v", err)
	}
	if len(stats.TopKCards) != 5 {
		t.Fatalf("len(TopKCards) = %d, want 5", len(stats.TopKCards))
	}
	if got := stats.TopKCards[0].Name; got != "TheBest" {
		t.Errorf("TopKCards[0] = %q, want %q (must be the single most expensive card across all 8, not just the first 5)", got, "TheBest")
	}
}

// newTestServiceWithSetInfo primes both caches — cards via newTestService, and
// set metadata via setc.reload, which (like cardsCache.reload) just takes a
// plain loader func, so no store/scryfall stub is needed here either.
func newTestServiceWithSetInfo(t *testing.T, cards []store.Card, sets map[string]scryfall.SetInfo) *Service {
	t.Helper()
	svc := newTestService(t, cards)
	if err := svc.setc.reload(context.Background(), func(ctx context.Context) (*map[string]scryfall.SetInfo, error) {
		return &sets, nil
	}); err != nil {
		t.Fatalf("prime set cache: %v", err)
	}
	return svc
}

func intPtr(i int) *int { return &i }

func TestGetSetInfoSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{
			{Name: "Sol Ring", Set: "M10", Number: "5"},
			{Name: "Bolt", Set: "M10", Number: "8"},
		},
		map[string]scryfall.SetInfo{
			"m10": {SetType: "core", PrintedSize: intPtr(10), IconSVGUri: "https://example.com/m10.svg"},
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("len(sets) = %d, want 1", len(sets))
	}
	want := SetCompletion{ImageURI: "https://example.com/m10.svg", Set: "M10", Owned: 2, Total: 10}
	if sets[0] != want {
		t.Errorf("sets[0] = %+v, want %+v", sets[0], want)
	}
}

func TestGetSetInfoErrLoadNotReadyError(t *testing.T) {
	svc := &Service{}
	_, err := svc.GetSetInfo(context.Background())
	if err == nil {
		t.Fatal("expected error when snapshot isn't loaded yet")
	}
}

func TestGetSetInfoEmptyCollectionSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{})

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 0 {
		t.Errorf("len(sets) = %d, want 0", len(sets))
	}
}

func TestGetSetInfoSetMetadataNotLoadedSuccess(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Bolt", Set: "M10", Number: "149"},
	}) // setc never primed — setmd stays nil

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 0 {
		t.Errorf("len(sets) = %d, want 0 (every set is skipped when set metadata hasn't loaded, even though the collection has real sets)", len(sets))
	}
}

func TestGetSetInfoUnknownSetSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{
			{Name: "Homebrew Card", Set: "ZZZ", Number: "1"},
		},
		map[string]scryfall.SetInfo{
			"m10": {SetType: "core", PrintedSize: intPtr(10)}, // ZZZ intentionally absent
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 0 {
		t.Errorf("len(sets) = %d, want 0 (ZZZ isn't in Scryfall's metadata — skipped, not errored)", len(sets))
	}
}

func TestGetSetInfoNonCompletableSetTypeSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{
			{Name: "Sol Ring", Set: "CMD", Number: "1"},
		},
		map[string]scryfall.SetInfo{
			"cmd": {SetType: "commander", PrintedSize: intPtr(100)},
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 0 {
		t.Errorf("len(sets) = %d, want 0 (commander sets aren't completable, should be dropped)", len(sets))
	}
}

func TestGetSetInfoPrintedSizeFallbackSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{{Name: "Bolt", Set: "M10", Number: "1"}},
		map[string]scryfall.SetInfo{
			"m10": {SetType: "core", PrintedSize: intPtr(15), CardCount: 999},
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("len(sets) = %d, want 1", len(sets))
	}
	if sets[0].Total != 15 {
		t.Errorf("Total = %d, want 15 (PrintedSize should win over CardCount when both are set)", sets[0].Total)
	}
}

func TestGetSetInfoCardCountFallbackSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{{Name: "Bolt", Set: "M10", Number: "1"}},
		map[string]scryfall.SetInfo{
			"m10": {SetType: "core", PrintedSize: nil, CardCount: 250},
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("len(sets) = %d, want 1", len(sets))
	}
	if sets[0].Total != 250 {
		t.Errorf("Total = %d, want 250 (should fall back to CardCount when PrintedSize is nil)", sets[0].Total)
	}
}

func TestGetSetInfoOutOfRangeCollectorNumberSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{
			{Name: "In Range", Set: "M10", Number: "5"},
			{Name: "Bonus Sheet Card", Set: "M10", Number: "999"}, // beyond the printed set
		},
		map[string]scryfall.SetInfo{
			"m10": {SetType: "core", PrintedSize: intPtr(10)},
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("len(sets) = %d, want 1", len(sets))
	}
	if sets[0].Owned != 1 {
		t.Errorf("Owned = %d, want 1 (the out-of-range collector number shouldn't count)", sets[0].Owned)
	}
}

func TestGetSetInfoUnknownTotalCountsAllSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{
			{Name: "Weird Numbering", Set: "XXX", Number: "9999"},
		},
		map[string]scryfall.SetInfo{
			"xxx": {SetType: "expansion", PrintedSize: nil, CardCount: 0}, // Total ends up 0 — unknown size
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("len(sets) = %d, want 1", len(sets))
	}
	if sets[0].Owned != 1 {
		t.Errorf("Owned = %d, want 1 (Total == 0 means no bounds check — every collector number counts)", sets[0].Owned)
	}
}

func TestGetSetInfoNonNumericCollectorNumberSuccess(t *testing.T) {
	svc := newTestServiceWithSetInfo(t,
		[]store.Card{
			{Name: "Promo Variant", Set: "M10", Number: "150a"},
		},
		map[string]scryfall.SetInfo{
			"m10": {SetType: "core", PrintedSize: intPtr(10)},
		},
	)

	sets, err := svc.GetSetInfo(context.Background())
	if err != nil {
		t.Fatalf("GetSetInfo: %v", err)
	}
	if len(sets) != 1 {
		t.Fatalf("len(sets) = %d, want 1", len(sets))
	}
	if sets[0].Owned != 0 {
		t.Errorf("Owned = %d, want 0 (non-numeric collector numbers are excluded, but the set itself still appears)", sets[0].Owned)
	}
}
func TestMatchesTrue(t *testing.T) {

	card := store.Card{Name: "Lightning Bolt", Set: "M10", Number: "1", Colors: []string{"R"}, Rarity: "common"}
	wantName := store.SearchFilter{Name: "Lightning Bolt"}
	wantSet := store.SearchFilter{Sets: []string{"M10"}}
	wantColors := store.SearchFilter{Colors: []string{"R"}}
	wantRarity := store.SearchFilter{Rarity: []string{"common"}}
	emptyFilter := store.SearchFilter{}
	assertName := matches(card, wantName)
	assertSet := matches(card, wantSet)
	assertColors := matches(card, wantColors)
	assertRarity := matches(card, wantRarity)
	assertEmpty := matches(card, emptyFilter)
	if !assertName {
		t.Errorf("Expected true for %s, based on parameter of %s", card.Name, wantName.Name)
	}
	if !assertSet {
		t.Errorf("Expected true for %s, based on parameter of %s", card.Set, wantSet.Sets[0])
	}
	if !assertColors {
		t.Errorf("Expected true for %v, based on parameter of %v", card.Colors, wantSet.Colors)
	}
	if !assertRarity {
		t.Errorf("Expected true for %s, based on parameter of %v", card.Rarity, wantSet.Rarity)
	}
	if !assertEmpty {
		t.Errorf("Expected true, as matches should happen with no filter")
	}
}
func TestMatchesFalse(t *testing.T) {
	card := store.Card{Name: "Lightning Bolt", Set: "M10", Number: "1", Colors: []string{"R"}, Rarity: "common"}
	wantName := store.SearchFilter{Name: "Arcane Signet"}
	wantSet := store.SearchFilter{Sets: []string{"M12"}}
	wantColors := store.SearchFilter{Colors: []string{"U"}}
	wantRarity := store.SearchFilter{Rarity: []string{"uncommon"}}
	assertName := matches(card, wantName)
	assertSet := matches(card, wantSet)
	assertColors := matches(card, wantColors)
	assertRarity := matches(card, wantRarity)
	if assertName {
		t.Errorf("Expected false for %s, based on parameter of %s", card.Name, wantName.Name)
	}
	if assertSet {
		t.Errorf("Expected false for %s, based on parameter of %s", card.Set, wantSet.Sets[0])
	}
	if assertColors {
		t.Errorf("Expected false for %v, based on parameter of %v", card.Colors, wantColors.Colors)
	}
	if assertRarity {
		t.Errorf("Expected false for %s, based on parameter of %v", card.Rarity, wantRarity.Rarity)
	}
}
func TestMatchingAllColorsCard(t *testing.T) {
	card := store.Card{Name: "Jodah", Set: "CMM", Number: "1", Colors: []string{"W", "U", "R", "G", "B"}, Rarity: "rare"}
	wantSingleColor := store.SearchFilter{Colors: []string{"R"}}
	wantMultipleColors := store.SearchFilter{Colors: []string{"W", "U"}}
	wantRarities := store.SearchFilter{Rarity: []string{"rare", "mythic"}}
	assertSingleColor := matches(card, wantSingleColor)
	assertMultipleColors := matches(card, wantMultipleColors)
	assertRarities := matches(card, wantRarities)
	if !assertSingleColor {
		t.Errorf("Expected true based on single color search of multi color card")
	}
	if !assertMultipleColors {
		t.Errorf("Expected true based on multiple color search for 5 color card")
	}
	if !assertRarities {
		t.Errorf("Expected true based on multiple rarity search for single rarity card")
	}
}
func TestBuildFilteredCardsSuccess(t *testing.T) {

}
func TestBuildFilteredCardsNoCards(t *testing.T) {

}
func TestBuildFilteredCardsNamesASC(t *testing.T) {

}
func TestBuildFilteredCardsNamesDESC(t *testing.T) {

}
func TestBuildFilteredCardsPricesASC(t *testing.T) {

}
func TestBuildFilteredCardsPricesDESC(t *testing.T) {

}
func TestBuildFilteredCardsSortUnspecified(t *testing.T) {

}
func TestBuildFilteredCardsCardsOfSamePrice(t *testing.T) {

}

func TestPaginateFirstPageNoNextTokenSuccess(t *testing.T) {
	cards := []store.Card{
		{Name: "Card1", Set: "SET", Number: "1"},
		{Name: "Card2", Set: "SET", Number: "2"},
		{Name: "Card3", Set: "SET", Number: "3"},
	}

	page, next, err := paginate(cards, 10, "")
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if len(page) != 3 {
		t.Errorf("len(page) = %d, want 3", len(page))
	}
	if next != "" {
		t.Errorf("next = %q, want empty (page_size covers the whole collection)", next)
	}
}

func TestPaginateEmptyTokenStartsFromBeginningSuccess(t *testing.T) {
	cards := []store.Card{
		{Name: "Card1", Set: "SET", Number: "1"},
		{Name: "Card2", Set: "SET", Number: "2"},
		{Name: "Card3", Set: "SET", Number: "3"},
	}

	page, _, err := paginate(cards, 1, "")
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if len(page) != 1 {
		t.Fatalf("len(page) = %d, want 1", len(page))
	}
	if page[0].Name != "Card1" {
		t.Errorf("page[0] = %q, want %q (empty token shouldn't skip the first card)", page[0].Name, "Card1")
	}
}

func TestPaginateWalksAllPagesWithoutGapOrDuplicateSuccess(t *testing.T) {
	cards := []store.Card{
		{Name: "Card1", Set: "SET", Number: "1"},
		{Name: "Card2", Set: "SET", Number: "2"},
		{Name: "Card3", Set: "SET", Number: "3"},
		{Name: "Card4", Set: "SET", Number: "4"},
		{Name: "Card5", Set: "SET", Number: "5"},
	}

	var got []store.Card
	token := ""
	for {
		page, next, err := paginate(cards, 2, token)
		if err != nil {
			t.Fatalf("paginate: %v", err)
		}
		got = append(got, page...)
		if next == "" {
			break
		}
		token = next
	}

	if len(got) != len(cards) {
		t.Fatalf("got %d cards across all pages, want %d", len(got), len(cards))
	}
	for i, c := range got {
		if c.Name != cards[i].Name {
			t.Errorf("page-walk[%d] = %s, want %s", i, c.Name, cards[i].Name)
		}
	}
}

func TestPaginateZeroOrNegativePageSizeMeansNoLimitSuccess(t *testing.T) {
	cards := []store.Card{
		{Name: "Card1", Set: "SET", Number: "1"},
		{Name: "Card2", Set: "SET", Number: "2"},
		{Name: "Card3", Set: "SET", Number: "3"},
	}

	for _, pageSize := range []int32{0, -1} {
		page, next, err := paginate(cards, pageSize, "")
		if err != nil {
			t.Fatalf("paginate(pageSize=%d): %v", pageSize, err)
		}
		if len(page) != 3 {
			t.Errorf("paginate(pageSize=%d): len(page) = %d, want 3", pageSize, len(page))
		}
		if next != "" {
			t.Errorf("paginate(pageSize=%d): next = %q, want empty", pageSize, next)
		}
	}
}

func TestPaginateInvalidTokenError(t *testing.T) {
	cards := []store.Card{
		{Name: "Card1", Set: "SET", Number: "1"},
	}

	_, _, err := paginate(cards, 10, "not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error from a corrupted page token, got nil")
	}
}

func TestDecodeCursorEmptyTokenSuccess(t *testing.T) {
	got, err := decodeCursor("")
	if err != nil {
		t.Fatalf("decodeCursor: %v", err)
	}
	if got != "" {
		t.Errorf("decodeCursor(\"\") = %q, want empty", got)
	}
}

func TestDecodeCursorInvalidBase64Error(t *testing.T) {
	_, err := decodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error decoding a corrupted token, got nil")
	}
}

func TestEncodeDecodeCursorRoundTripSuccess(t *testing.T) {
	want := "Lightning Bolt-M10-149-nonfoil"
	got, err := decodeCursor(encodeCursor(want))
	if err != nil {
		t.Fatalf("decodeCursor: %v", err)
	}
	if got != want {
		t.Errorf("round trip = %q, want %q", got, want)
	}
}
func TestSearchSucces(t *testing.T) {
	svc := newTestService(t, []store.Card{
		{Name: "Lightning Bolt", Set: "M10", Number: "1", Colors: []string{"R"}, Prices: store.Prices{USD: "1.00"}},
		{Name: "Shivan Dragon", Set: "M10", Number: "2", Colors: []string{"R"}, Prices: store.Prices{USD: "5.00"}},
		{Name: "Sol Ring", Set: "C21", Number: "3", Colors: []string{}, Prices: store.Prices{USD: "2.00"}},
	})
	got, next, err := svc.SearchCards(context.Background(), "", nil, []string{"R"}, nil, 10, "", PRICE_DESC)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Name != "Shivan Dragon" || got[1].Name != "Lightning Bolt" {
		t.Errorf("got = [%s, %s], want [Shivan Dragon, Lightning Bolt]", got[0].Name, got[1].Name)
	}
	if next != "" {
		t.Errorf("next = %q, want empty (page_size covers both results)", next)
	}
}
func TestSearchEmptyCollection(t *testing.T) {
	svc := newTestService(t, []store.Card{})

	got, next, err := svc.SearchCards(context.Background(), "", nil, nil, nil, 10, "", SORT_UNSPECIFIED)
	if err != nil {
		t.Fatalf("SearchCards: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
	if next != "" {
		t.Errorf("next = %q, want empty", next)
	}
}
func TestSearchErrLoadingSnapshot(t *testing.T) {
	svc := &Service{}
	_, _, err := svc.SearchCards(context.Background(), "Name", []string{}, []string{}, []string{}, 2, "", SORT_UNSPECIFIED)
	if err == nil {
		t.Fatal("expected error when snapshot isn't loaded yet")
	}
}
