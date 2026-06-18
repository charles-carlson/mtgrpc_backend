package ingest

import (
	"testing"
)

func TestParseCSVRow(t *testing.T) {
	tests := []struct {
		name    string
		row     []string
		want    parsedCard
		wantErr bool
	}{
		{
			name: "basic card",
			row:  []string{"2", "Lightning Bolt", "M10", "149"},
			want: parsedCard{count: 2, name: "Lightning Bolt", set: "M10", number: "149"},
		},
		{
			name: "multi-word name",
			row:  []string{"1", "Talisman of Dominance", "MRD", "220"},
			want: parsedCard{count: 1, name: "Talisman of Dominance", set: "MRD", number: "220"},
		},
		{
			name: "trims whitespace",
			row:  []string{" 3 ", " Black Lotus ", " LEA ", " 232 "},
			want: parsedCard{count: 3, name: "Black Lotus", set: "LEA", number: "232"},
		},
		{
			name:    "too few columns",
			row:     []string{"1", "Lightning Bolt", "M10"},
			wantErr: true,
		},
		{
			name:    "invalid quantity",
			row:     []string{"x", "Lightning Bolt", "M10", "149"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card, err := parseCSVRow(tt.row)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseCSVRow(%v) error = %v, wantErr %v", tt.row, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if card.Count != tt.want.count || card.Name != tt.want.name || card.Set != tt.want.set || card.Number != tt.want.number {
				t.Errorf("got {%d %q %q %q}, want {%d %q %q %q}",
					card.Count, card.Name, card.Set, card.Number,
					tt.want.count, tt.want.name, tt.want.set, tt.want.number)
			}
		})
	}
}
