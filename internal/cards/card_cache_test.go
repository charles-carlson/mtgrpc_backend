package cards

import (
	"backend_nonsense/internal/store"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCardCache_ConcurrentReadWrite(t *testing.T) {
	var c cardsCache // zero value is ready; do NOT copy it after this

	var loadCalls atomic.Int64
	load := func(ctx context.Context) ([]store.Card, error) {
		loadCalls.Add(1)
		return []store.Card{
			{Name: "Bolt", Set: "M10", Number: "149"},
			{Name: "Sol Ring", Set: "C21", Number: "263"},
		}, nil
	}

	// Prime once so readers never observe a nil snapshot.
	if err := c.reload(context.Background(), load); err != nil {
		t.Fatalf("prime reload: %v", err) // t.Fatalf OK here — main test goroutine
	}

	const (
		readers    = 8
		writers    = 4
		iterations = 1000
	)

	var wg sync.WaitGroup
	start := make(chan struct{}) // gate: release everyone at once for max contention

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				snap := c.current()
				if snap == nil {
					t.Errorf("current() returned nil after prime") // t.Errorf, NOT Fatalf
					return
				}
				// Actually touch the data so -race observes the reads.
				_ = len(snap.sets)
				_ = snap.byKey["Bolt-M10-149"]
			}
		}()
	}

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < iterations; j++ {
				if err := c.reload(context.Background(), load); err != nil {
					t.Errorf("reload: %v", err)
					return
				}
			}
		}()
	}

	close(start) // go!
	wg.Wait()

	if loadCalls.Load() == 0 {
		t.Fatal("expected load to be called")
	}
}
func TestCardCache_ReloadErrorKeepsPrevious(t *testing.T) {
	var c cardsCache
	good := func(ctx context.Context) ([]store.Card, error) {
		return []store.Card{{Name: "Bolt", Set: "M10", Number: "149"}}, nil
	}
	if err := c.reload(context.Background(), good); err != nil {
		t.Fatalf("prime: %v", err)
	}
	prev := c.current()

	bad := func(ctx context.Context) ([]store.Card, error) {
		return nil, errors.New("dynamo down")
	}
	if err := c.reload(context.Background(), bad); err == nil {
		t.Fatal("expected error from failed reload")
	}
	if c.current() != prev {
		t.Error("failed reload replaced the snapshot; previous one should be preserved")
	}
}
