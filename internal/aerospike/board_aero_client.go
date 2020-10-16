package aerospike

import (
	"errors"
	"fmt"

	as "github.com/aerospike/aerospike-client-go"
)

// compile time check - ensure that BoardAeroClients implements Client interface
var _ Client = (*BoardAeroClient)(nil)

// aerospike data model (namespace, set, record, bin, ...) infos:
// https://aerospike.com/docs/architecture/data-model.html
type BoardAeroClient struct {
	namespace  string
	aeroClient *as.Client
}

func NewBoardAeroClient(aeroClient *as.Client, namespace string) (*BoardAeroClient, error) {
	if aeroClient == nil {
		return nil, errors.New("aero client is nil")
	}
	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}

	return &BoardAeroClient{
		namespace:  namespace,
		aeroClient: aeroClient,
	}, nil
}

func (bc *BoardAeroClient) Put(set, key string, binMap AeroBinMap) error {
	aeroKey, err := as.NewKey(bc.namespace, set, key)
	if err != nil {
		return err
	}

	// TODO: create the right policy ?
	if err = bc.aeroClient.Put(nil, aeroKey, as.BinMap(binMap)); err != nil {
		return err
	}

	return nil
}

func (bc *BoardAeroClient) QueryByRange(set string, index string, from, to int64) ([]AeroBinMap, error) {
	rangeFilterStt := &as.Statement{
		Namespace: bc.namespace,
		SetName:   set,
		IndexName: index,
		Filter:    as.NewRangeFilter(index, from, to),
	}

	recordSet, err := bc.aeroClient.Query(nil, rangeFilterStt)
	if err != nil {
		return nil, fmt.Errorf("failed to query aero for range filter set: %w", err)
	}

	return bc.RecordSet2AeroBinMaps(recordSet)
}

func (bc *BoardAeroClient) ScanAll(set string) ([]AeroBinMap, error) {
	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = true

	recordSet, err := bc.aeroClient.ScanAll(spolicy, bc.namespace, set)
	if err != nil {
		return nil, err
	}

	return bc.RecordSet2AeroBinMaps(recordSet)
}

func (bc *BoardAeroClient) CountAll(set string) (int, error) {
	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = false

	recs, err := bc.aeroClient.ScanAll(spolicy, bc.namespace, set)
	if err != nil {
		return -1, err
	}

	count := 0
	for _ = range recs.Results() {
		count++
	}

	return count, nil
}

func (bc *BoardAeroClient) IsConnected() bool {
	if bc.aeroClient == nil {
		return false
	}
	return bc.aeroClient.IsConnected()
}

func (bc *BoardAeroClient) Close() {
	if bc.aeroClient == nil || bc.aeroClient.IsConnected() {
		return
	}
	bc.aeroClient.Close()
}

func (bc *BoardAeroClient) RecordSet2AeroBinMaps(recordSet *as.Recordset) ([]AeroBinMap, error) {
	var binMap []AeroBinMap
	for rec := range recordSet.Results() {
		if rec.Err != nil {
			return nil, fmt.Errorf("query by range, record error: %s", rec.Err)
		}
		aeroBin := make(map[string]interface{})
		for k, v := range rec.Record.Bins {
			aeroBin[k] = v
		}
		binMap = append(binMap, aeroBin)
	}

	return binMap, nil
}
