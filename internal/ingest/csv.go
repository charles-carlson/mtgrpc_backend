package ingest

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/store"
)

// RunFromCSV reads a CSV file with columns: quantity, name, set, number[, finish].
// The finish column is optional (defaults to nonfoil). The first row is treated
// as a header and skipped.
func RunFromCSV(ctx context.Context, path string, svc *cards.Service) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	rows, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("read csv: %w", err)
	}
	if len(rows) < 2 {
		return nil // empty or header-only
	}

	for i, row := range rows[1:] {
		lineNum := i + 2 // account for skipped header
		card, err := parseCSVRow(row)
		if err != nil {
			return fmt.Errorf("row %d: %w", lineNum, err)
		}

		if err := svc.AddCard(ctx, card); err != nil {
			return fmt.Errorf("row %d: add card %q: %w", lineNum, card.Name, err)
		}
		log.Printf("ingested %q (%s/%s)", card.Name, card.Set, card.Number)
	}

	return nil
}

func parseCSVRow(row []string) (store.Card, error) {
	if len(row) < 4 {
		return store.Card{}, fmt.Errorf("expected at least 4 columns, got %d", len(row))
	}

	count, err := strconv.Atoi(strings.TrimSpace(row[0]))
	if err != nil {
		return store.Card{}, fmt.Errorf("invalid quantity %q: %w", row[0], err)
	}

	finishRaw := ""
	if len(row) >= 5 {
		finishRaw = row[4]
	}
	finish, err := normalizeFinish(finishRaw)
	if err != nil {
		return store.Card{}, err
	}

	return store.Card{
		Count:  count,
		Name:   strings.TrimSpace(row[1]),
		Set:    strings.TrimSpace(row[2]),
		Number: strings.TrimSpace(row[3]),
		Finish: finish,
	}, nil
}
