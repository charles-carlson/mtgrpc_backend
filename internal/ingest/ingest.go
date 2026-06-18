package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/store"
)

// RunFile dispatches to the correct ingest function based on file extension.
// Supports .json (Manabox export) and .txt (plain-text format).
func RunFile(ctx context.Context, path string, svc *cards.Service) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return runJSON(ctx, path, svc)
	case ".txt":
		return RunFromText(ctx, path, svc)
	case ".csv":
		return RunFromCSV(ctx, path, svc)
	default:
		return fmt.Errorf("unsupported file type %q: use .json, .txt, or .csv", filepath.Ext(path))
	}
}

func runJSON(ctx context.Context, path string, svc *cards.Service) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var entries []store.Card
	if err := json.NewDecoder(f).Decode(&entries); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}

	for _, card := range entries {
		if err := svc.AddCard(ctx, card); err != nil {
			return fmt.Errorf("add card %q: %w", card.Name, err)
		}
		log.Printf("ingested %q (%s/%s)", card.Name, card.Set, card.Number)
	}

	return nil
}
