package cards

import (
	"backend_nonsense/internal/scryfall"
	"context"
	"sync/atomic"
)

type setCache struct {
	v atomic.Pointer[map[string]scryfall.SetInfo]
}

func (s *setCache) current() *map[string]scryfall.SetInfo { return s.v.Load() }
func (s *setCache) reload(ctx context.Context, load func(context.Context) (*map[string]scryfall.SetInfo, error)) error {
	sets, err := load(ctx)
	if err != nil {
		return err
	}
	s.v.Store(sets)
	return nil
}
