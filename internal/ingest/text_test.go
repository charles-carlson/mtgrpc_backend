package ingest

import (
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		input   string
		want    parsedCard
		wantErr bool
	}{
		{
			input: "2 Lightning Bolt M10 149",
			want:  parsedCard{count: 2, name: "Lightning Bolt", set: "M10", number: "149"},
		},
		{
			input: "1 Arcane Signet ELD 331",
			want:  parsedCard{count: 1, name: "Arcane Signet", set: "ELD", number: "331"},
		},
		{
			input: "4 Talisman of Dominance MRD 220",
			want:  parsedCard{count: 4, name: "Talisman of Dominance", set: "MRD", number: "220"},
		},
		{
			input:   "bad input",
			wantErr: true,
		},
		{
			input:   "abc Lightning Bolt M10 149",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			card, err := parseLine(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseLine(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
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

type parsedCard struct {
	count  int
	name   string
	set    string
	number string
}
