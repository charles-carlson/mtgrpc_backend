package cards

import (
	"backend_nonsense/internal/store"
	"fmt"
	"slices"
	"testing"
)

func TestBuildSnapshot(t *testing.T) {
	tests := []struct {
		name     string
		input    []store.Card
		wantSets []string
	}{
		{
			name:     "empty input",
			input:    nil,
			wantSets: []string{},
		},
		{
			name: "dedup and sort sets",
			input: []store.Card{
				{Name: "Sol Ring", Set: "M10", Number: "263"},
				{Name: "Bolt", Set: "A25", Number: "141"},
				{Name: "Bolt", Set: "M10", Number: "149"},
			},
			wantSets: []string{"A25", "M10"},
		},
		{
			name: "single card dedup and sort sets",
			input: []store.Card{
				{Name: "Counterspell", Set: "CMM", Number: "081"},
			},
			wantSets: []string{"CMM"},
		},
		{
			name: "same cards but different sets",
			input: []store.Card{
				{Name: "Counterspell", Set: "6ED", Number: "61"},
				{Name: "Counterspell", Set: "MH2", Number: "267"},
				{Name: "Counterspell", Set: "STA", Number: "015"},
				{Name: "Counterspell", Set: "CMM", Number: "081"},
				{Name: "Counterspell", Set: "DSC", Number: "114"},
				{Name: "Counterspell", Set: "ICE", Number: "64"},
			},
			wantSets: []string{"6ED", "CMM", "DSC", "ICE", "MH2", "STA"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snap := buildSnapshot(tc.input)
			if !slices.Equal(snap.sets, tc.wantSets) {
				t.Errorf("sets = %v, want %v", snap.sets, tc.wantSets)
			}
			if !(len(snap.allCards) == len(tc.input)) {
				t.Errorf("allCards = %d, want %d", len(snap.allCards), len(tc.input))
			}
			if len(snap.byKey) != len(tc.input) {
				t.Errorf("byKey has %d entries, want %d (key collision?)", len(snap.byKey), len(tc.input))
			}
			for _, card := range tc.input {
				expectedKey := fmt.Sprintf("%s-%s-%s", card.Name, card.Set, card.Number)
				if _, ok := snap.byKey[expectedKey]; !ok {
					t.Errorf("expected=%s, does not exist", expectedKey)
				}
			}
		})
	}
}
