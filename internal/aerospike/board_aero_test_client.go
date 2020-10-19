package aerospike

import (
	"errors"
	"fmt"
	"sync"
)

// compile time check - ensure that BoardAeroClients implements Client interface
var _ Client = (*BoardAeroTestClient)(nil)

type BoardAeroTestClient struct {
	AeroBinMaps map[string]AeroBinMap
	mutex       sync.Mutex

	IsConnectedValue bool
}

func NewBoardAeroTestClient() *BoardAeroTestClient {
	return &BoardAeroTestClient{
		AeroBinMaps:      make(map[string]AeroBinMap),
		IsConnectedValue: true,
	}
}

func NewBoardAeroTestClientWithBins(aeroBinMaps map[string]AeroBinMap) *BoardAeroTestClient {
	return &BoardAeroTestClient{
		AeroBinMaps:      aeroBinMaps,
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
	tc.AeroBinMaps[key] = binMap
	return nil
}

func (tc *BoardAeroTestClient) Delete(key string) (bool, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if key == "" {
		return false, errors.New("empty key")
	}

	_, found := tc.AeroBinMaps[key]
	if !found {
		return false, nil
	}

	delete(tc.AeroBinMaps, key)

	return true, nil
}

func (tc *BoardAeroTestClient) QueryByRange(index string, from, to int64) ([]AeroBinMap, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	var binMaps []AeroBinMap
	for _, binMap := range tc.AeroBinMaps {
		val, indexFound := binMap[index]
		if !indexFound {
			continue
		}

		var valInt64 int64
		switch valType := val.(type) {
		case int:
			valInt64 = int64(val.(int))
		case int64:
			valInt64 = val.(int64)
		default:
			fmt.Printf("aero test - query by range, unknown type %T!\n", valType)
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
	for _, binMap := range tc.AeroBinMaps {
		binMaps = append(binMaps, binMap)
	}
	return binMaps, nil
}

func (tc *BoardAeroTestClient) CountAll() (int, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	if tc.AeroBinMaps == nil {
		return -1, errors.New("nil aero bin maps")
	}
	return len(tc.AeroBinMaps), nil
}

func (tc *BoardAeroTestClient) IsConnected() bool {
	return tc.IsConnectedValue
}

func (tc *BoardAeroTestClient) Close() {
	// nop
}
