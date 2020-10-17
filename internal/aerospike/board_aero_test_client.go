package aerospike

import (
	"errors"
	"sync"
)

// compile time check - ensure that BoardAeroClients implements Client interface
var _ Client = (*BoardAeroTestClient)(nil)

type BoardAeroTestClient struct {
	aeroBinMaps map[string]AeroBinMap
	mutex       sync.Mutex
}

func NewBoardAeroTestClient() *BoardAeroTestClient {
	return &BoardAeroTestClient{
		aeroBinMaps: make(map[string]AeroBinMap),
	}
}

func (tc *BoardAeroTestClient) Put(key string, binMap AeroBinMap) error {
	panic("implement me")
}

func (tc *BoardAeroTestClient) Delete(key string) (bool, error) {
	panic("implement me")
}

func (tc *BoardAeroTestClient) QueryByRange(index string, from, to int64) ([]AeroBinMap, error) {
	panic("implement me")
}

func (tc *BoardAeroTestClient) ScanAll() ([]AeroBinMap, error) {
	panic("implement me")
}

func (tc *BoardAeroTestClient) CountAll() (int, error) {
	if tc.aeroBinMaps == nil {
		return -1, errors.New("nil aero bin maps")
	}
	return len(tc.aeroBinMaps), nil
}

func (tc *BoardAeroTestClient) IsConnected() bool {
	panic("implement me")
}

func (tc *BoardAeroTestClient) Close() {
	// nop
}
