package aerospike

import (
	"sync"
)

// compile time check - ensure that BoardAeroClients implements Client interface
var _ Client = (*BoardAeroTestClient)(nil)

type BoardAeroTestClient struct {
	ReadBinMaps map[string]AeroBinMap
	mutex       sync.Mutex
}

func (tc *BoardAeroTestClient) Put(set, key string, binMap AeroBinMap) error {
	panic("implement me")
}

func (tc *BoardAeroTestClient) QueryByRange(set string, index string, from, to int64) ([]AeroBinMap, error) {
	panic("implement me")
}

func (tc *BoardAeroTestClient) ScanAll(set string) ([]AeroBinMap, error) {
	panic("implement me")
}

func (tc *BoardAeroTestClient) CountAll(set string) (int, error) {
	panic("implement me")
}

func (tc *BoardAeroTestClient) IsConnected() bool {
	panic("implement me")
}

func (tc *BoardAeroTestClient) Close() {
	// nop
}
