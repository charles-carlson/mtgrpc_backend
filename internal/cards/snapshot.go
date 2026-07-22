package cards

import (
	"backend_nonsense/internal/store"
	"fmt"
	"slices"
	"strings"
)

// future builds might be required for decklist service,
type snapshot struct {
	allCards []store.Card          //all cards
	byKey    map[string]store.Card //name->set->number
	sets     []string              // list of sets
}

func keyOf(c store.Card) string {
	return fmt.Sprintf("%s-%s-%s-%s", c.Name, c.Set, c.Number, c.Finish)
}

func buildSnapshot(cards []store.Card) *snapshot {
	// init variables
	byKey := make(map[string]store.Card)
	sets := make(map[string]struct{})
	//single pass to build map and set hashmap
	for _, card := range cards {
		key := keyOf(card)
		byKey[key] = card
		if card.Set != "" {
			sets[card.Set] = struct{}{}
		}
	}
	//flatten map and sort into string array
	flattenSets := make([]string, 0, len(sets))
	for key := range sets {
		flattenSets = append(flattenSets, key)
	}
	slices.Sort(flattenSets)
	//Sort cards by their key
	slices.SortFunc(cards, func(a, b store.Card) int { return strings.Compare(keyOf(a), keyOf(b)) })
	//return pointer of snapshot
	return &snapshot{
		allCards: cards,
		byKey:    byKey,
		sets:     flattenSets,
	}
}
