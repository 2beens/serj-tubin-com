package visitor_board

import (
	"sync"
)

var _ Cache = (*BoardTestCache)(nil)

type TestFuncCall int

const (
	FuncGetMiss TestFuncCall = iota
	FuncGetHit
	FuncSet
	FuncClear
)

type BoardTestCache struct {
	cache            map[interface{}]interface{}
	FunctionCallsLog []TestFuncCall
	mutex            sync.Mutex
}

func NewBoardTestCache() *BoardTestCache {
	return &BoardTestCache{
		cache:            make(map[interface{}]interface{}),
		FunctionCallsLog: []TestFuncCall{},
	}
}

func (btc *BoardTestCache) ElementsCount() int {
	return len(btc.cache)
}

func (btc *BoardTestCache) ClearFunctionCallsLog() {
	btc.FunctionCallsLog = []TestFuncCall{}
}

// ristretto.Cache can return nil and true pair, maybe test that case in the Board too
func (btc *BoardTestCache) Get(key interface{}) (interface{}, bool) {
	btc.mutex.Lock()
	defer btc.mutex.Unlock()

	if btc.cache == nil {
		panic("cache is nil")
	}

	if val, ok := btc.cache[key]; ok {
		btc.FunctionCallsLog = append(btc.FunctionCallsLog, FuncGetHit)
		return val, true
	}

	btc.FunctionCallsLog = append(btc.FunctionCallsLog, FuncGetMiss)
	return nil, false
}

func (btc *BoardTestCache) Set(key, value interface{}, cost int64) bool {
	btc.mutex.Lock()
	defer btc.mutex.Unlock()

	if btc.cache == nil {
		panic("cache is nil")
	}

	btc.FunctionCallsLog = append(btc.FunctionCallsLog, FuncSet)
	btc.cache[key] = value

	return true
}

func (btc *BoardTestCache) Clear() {
	btc.mutex.Lock()
	defer btc.mutex.Unlock()

	if btc.cache == nil {
		panic("cache is nil")
	}

	btc.FunctionCallsLog = append(btc.FunctionCallsLog, FuncClear)
	btc.cache = make(map[interface{}]interface{})
}
