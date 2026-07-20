package cards

import (
	"backend_nonsense/internal/store"
	"context"
	"sync/atomic"
)

type cardsCache struct {
	v atomic.Pointer[snapshot]
}

// Returns pointer of current build of snapshot
func (c *cardsCache) current() *snapshot { return c.v.Load() }

// Generates new pointer of snapshot with updated cards
func (c *cardsCache) reload(ctx context.Context, load func(context.Context) ([]store.Card, error)) error {
	cards, err := load(ctx)
	if err != nil {
		return err
	}
	c.v.Store(buildSnapshot(cards))
	return nil
}
