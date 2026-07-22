package ingest

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"backend_nonsense/internal/cards"
	"backend_nonsense/internal/store"
)

// RunFromText reads a plain-text export where each line is:
//
//	<quantity> <name> <set> <number> [finish]
//
// Name may contain spaces. The optional trailing finish (nonfoil/foil/etched)
// is detected by value; set and number are the two tokens before it (or the
// last two when no finish is given).
func RunFromText(ctx context.Context, path string, svc *cards.Service) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		card, err := parseLine(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}

		if err := svc.AddCard(ctx, card); err != nil {
			return fmt.Errorf("line %d: add card %q: %w", lineNum, card.Name, err)
		}
		log.Printf("ingested %q (%s/%s)", card.Name, card.Set, card.Number)
	}

	return scanner.Err()
}

// parseLine parses: <quantity> <name...> <set> <number>
func parseLine(line string) (store.Card, error) {
	tokens := strings.Fields(line)
	if len(tokens) < 4 {
		return store.Card{}, fmt.Errorf("expected at least 4 fields, got %d: %q", len(tokens), line)
	}

	count, err := strconv.Atoi(tokens[0])
	if err != nil {
		return store.Card{}, fmt.Errorf("invalid quantity %q: %w", tokens[0], err)
	}

	// optional trailing finish token; when present, set/number sit just before it.
	end := len(tokens) // exclusive end of the set/number region
	finish := "nonfoil"
	if len(tokens) >= 5 && isFinish(tokens[end-1]) {
		f, err := normalizeFinish(tokens[end-1])
		if err != nil {
			return store.Card{}, err
		}
		finish = f
		end--
	}

	number := tokens[end-1]
	set := tokens[end-2]
	name := strings.Join(tokens[1:end-2], " ")

	return store.Card{
		Count:  count,
		Name:   name,
		Set:    set,
		Number: number,
		Finish: finish,
	}, nil
}
