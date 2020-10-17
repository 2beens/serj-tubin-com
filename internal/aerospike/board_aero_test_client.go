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

	IsConnectedValue bool
}

func NewBoardAeroTestClient() *BoardAeroTestClient {
	return &BoardAeroTestClient{
		aeroBinMaps:      make(map[string]AeroBinMap),
		IsConnectedValue: true,
	}
}

func NewBoardAeroTestClientWithBins(aeroBinMaps map[string]AeroBinMap) *BoardAeroTestClient {
	return &BoardAeroTestClient{
		aeroBinMaps:      aeroBinMaps,
		IsConnectedValue: true,
	}
}

func (tc *BoardAeroTestClient) Put(key string, binMap AeroBinMap) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	switch {
	case key == "":
		return errors.New("empty key")
	case binMap == nil:
		return errors.New("nil bin map")
	}
	tc.aeroBinMaps[key] = binMap
	return nil
}

func (tc *BoardAeroTestClient) Delete(key string) (bool, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if key == "" {
		return false, errors.New("empty key")
	}

	_, found := tc.aeroBinMaps[key]
	if !found {
		return false, nil
	}

	delete(tc.aeroBinMaps, key)

	return true, nil
}

func (tc *BoardAeroTestClient) QueryByRange(index string, from, to int64) ([]AeroBinMap, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	var binMaps []AeroBinMap
	for _, binMap := range tc.aeroBinMaps {
		val, indexFound := binMap[index]
		if !indexFound {
			continue
		}
		valInt64, ok := val.(int64)
		if !ok {
			continue
		}
		if valInt64 >= from && valInt64 <= to {
			binMaps = append(binMaps, binMap)
		}
	}
	return binMaps, nil
}

func (tc *BoardAeroTestClient) ScanAll() ([]AeroBinMap, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	var binMaps []AeroBinMap
	for _, binMap := range tc.aeroBinMaps {
		binMaps = append(binMaps, binMap)
	}
	return binMaps, nil
}

func (tc *BoardAeroTestClient) CountAll() (int, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if tc.aeroBinMaps == nil {
		return -1, errors.New("nil aero bin maps")
	}
	return len(tc.aeroBinMaps), nil
}

func (tc *BoardAeroTestClient) IsConnected() bool {
	return tc.IsConnectedValue
}

func (tc *BoardAeroTestClient) Close() {
	// nop
}
