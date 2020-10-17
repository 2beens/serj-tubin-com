package cache

import "sync"

var _ Cache = (*BoardTestCache)(nil)

type BoardTestCache struct {
	cache map[interface{}]interface{}
	mutex sync.Mutex
}

func NewBoardTestCache() *BoardTestCache {
	return &BoardTestCache{
		cache: make(map[interface{}]interface{}),
	}
}

// ristretto.Cache can return nil and true pair, maybe test that case in the Board too
func (btc *BoardTestCache) Get(key interface{}) (interface{}, bool) {
	btc.mutex.Lock()
	defer btc.mutex.Unlock()

	if btc.cache == nil {
		panic("cache is nil")
	}
	if val, ok := btc.cache[key]; ok {
		return val, true
	}
	return nil, false
}

func (btc *BoardTestCache) Set(key, value interface{}, cost int64) bool {
	btc.mutex.Lock()
	defer btc.mutex.Unlock()

	if btc.cache == nil {
		panic("cache is nil")
	}
	btc.cache[key] = value
	return true
}

func (btc *BoardTestCache) Clear() {
	btc.mutex.Lock()
	defer btc.mutex.Unlock()

	if btc.cache == nil {
		panic("cache is nil")
	}
	btc.cache = make(map[interface{}]interface{})
}
