package aerospike

import (
	"errors"
	"fmt"
	"strconv"

	as "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

// compile time check - ensure that BoardAeroClients implements Client interface
var _ Client = (*BoardAeroClient)(nil)

var (
	ErrAeroClientNil          = errors.New("aero client is nil")
	ErrAeroClientNotConnected = errors.New("aero client is not connected")
	ErrEmptyNamespace         = errors.New("namespace cannot be empty")
)

// aerospike data model (namespace, set, record, bin, ...) infos:
// https://aerospike.com/docs/architecture/data-model.html
type BoardAeroClient struct {
	namespace  string
	set        string
	aeroClient *as.Client
}

func NewBoardAeroClient(aeroClient *as.Client, namespace, set string) (*BoardAeroClient, error) {
	if aeroClient == nil {
		return nil, ErrAeroClientNil
	}
	if namespace == "" {
		return nil, ErrEmptyNamespace
	}

	return &BoardAeroClient{
		namespace:  namespace,
		set:        set,
		aeroClient: aeroClient,
	}, nil
}

func (bc *BoardAeroClient) Put(key string, binMap AeroBinMap) error {
	messageId, err := strconv.Atoi(key)
	if err != nil {
		return errors.New("failed to parse message id")
	}

	aeroKey, err := as.NewKey(bc.namespace, bc.set, messageId)
	if err != nil {
		return err
	}

	// TODO: create the right policy ?
	if err = bc.aeroClient.Put(nil, aeroKey, as.BinMap(binMap)); err != nil {
		return err
	}

	return nil
}

func (bc *BoardAeroClient) Delete(key string) (bool, error) {
	messageId, err := strconv.Atoi(key)
	if err != nil {
		return false, errors.New("failed to parse message id")
	}

	aeroKey, err := as.NewKey(bc.namespace, bc.set, messageId)
	if err != nil {
		return false, err
	}

	exists, err := bc.aeroClient.Exists(nil, aeroKey)
	if err != nil {
		return false, fmt.Errorf("failed to check key existance on aerospike: %w", err)
	} else if !exists {
		return false, errors.New("record does not exist")
	}

	removed, err := bc.aeroClient.Delete(nil, aeroKey)
	if err != nil {
		return false, fmt.Errorf("failed to run delete on aerospike: %w", err)
	}

	log.Tracef("message [%v] deleted: %t", aeroKey.String(), removed)

	return removed, nil
}

func (bc *BoardAeroClient) QueryByRange(index string, from, to int64) ([]AeroBinMap, error) {
	rangeFilterStt := &as.Statement{
		Namespace: bc.namespace,
		SetName:   bc.set,
		IndexName: index,
		Filter:    as.NewRangeFilter(index, from, to),
	}

	recordSet, err := bc.aeroClient.Query(nil, rangeFilterStt)
	if err != nil {
		return nil, fmt.Errorf("failed to query aero for range filter set: %w", err)
	}

	return bc.RecordSet2AeroBinMaps(recordSet)
}

func (bc *BoardAeroClient) ScanAll() ([]AeroBinMap, error) {
	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = true

	recordSet, err := bc.aeroClient.ScanAll(spolicy, bc.namespace, bc.set)
	if err != nil {
		return nil, err
	}

	return bc.RecordSet2AeroBinMaps(recordSet)
}

func (bc *BoardAeroClient) CountAll() (int, error) {
	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = false

	recs, err := bc.aeroClient.ScanAll(spolicy, bc.namespace, bc.set)
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
