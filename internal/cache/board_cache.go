package cache

import (
	"fmt"

	"github.com/dgraph-io/ristretto"
)

var _ Cache = (*BoardCache)(nil)

type BoardCache struct {
	cache *ristretto.Cache
}

func NewBoardCache() (*BoardCache, error) {
	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 28, // maximum cost of cache (~268M)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create ristretto cache: %s", err)
	}

	return &BoardCache{
		cache: c,
	}, nil
}

func (bc *BoardCache) Get(key interface{}) (interface{}, bool) {
	// TODO: check if ristretto function calls need to be mutex'ed
	return bc.cache.Get(key)
}

func (bc *BoardCache) Set(key, value interface{}, cost int64) bool {
	return bc.cache.Set(key, value, cost)
}

func (bc *BoardCache) Clear() {
	bc.cache.Clear()
}
