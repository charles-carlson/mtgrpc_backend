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
		finish, err := normalizeFinish(card.Finish)
		if err != nil {
			return fmt.Errorf("card %q: %w", card.Name, err)
		}
		card.Finish = finish
		if err := svc.AddCard(ctx, card); err != nil {
			return fmt.Errorf("add card %q: %w", card.Name, err)
		}
		log.Printf("ingested %q (%s/%s [%s])", card.Name, card.Set, card.Number, card.Finish)
	}

	return nil
}

// normalizeFinish lowercases and validates a finish, defaulting empty to
// "nonfoil". Accepts Manabox's "normal" as an alias for "nonfoil".
func normalizeFinish(f string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "", "normal", "nonfoil":
		return "nonfoil", nil
	case "foil":
		return "foil", nil
	case "etched":
		return "etched", nil
	default:
		return "", fmt.Errorf("invalid finish %q (want nonfoil, foil, or etched)", f)
	}
}

// isFinish reports whether a token is a recognized finish value — used to
// detect an optional trailing finish in the space-delimited text format.
func isFinish(s string) bool {
	switch strings.ToLower(s) {
	case "nonfoil", "foil", "etched", "normal":
		return true
	}
	return false
}
