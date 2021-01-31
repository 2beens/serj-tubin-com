package aerospike

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	as "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

// compile time check - ensure that BoardAeroClients implements Client interface
var _ Client = (*BoardAeroClient)(nil)

var (
	ErrAeroClientNil          = errors.New("aero client is nil")
	ErrAeroClientNotConnected = errors.New("aero client is not connected")
	ErrEmptyNamespace         = errors.New("namespace cannot be empty")
	ErrEmptySet               = errors.New("set cannot be empty")
)

// aerospike data model (namespace, set, record, bin, ...) infos:
// https://aerospike.com/docs/architecture/data-model.html
type BoardAeroClient struct {
	host string
	port int

	namespace   string
	set         string
	metaDataSet string // keep things like ID counter here, etc.
	aeroClient  *as.Client

	isConnecting bool
	mutex        sync.RWMutex
}

func NewBoardAeroClient(host string, port int, namespace, set string) (*BoardAeroClient, error) {
	log.Debugf("connecting to aerospike server %s:%d [namespace:%s, set:%s] ...",
		host, port, namespace, set)

	if set == "" {
		return nil, ErrEmptySet
	}
	if namespace == "" {
		return nil, ErrEmptyNamespace
	}

	bc := &BoardAeroClient{
		host:        host,
		port:        port,
		namespace:   namespace,
		set:         set,
		metaDataSet: set + "-metadata",
	}

	go func() {
		if err := bc.CheckConnection(); err != nil {
			log.Errorln(err)
		}
	}()

	return bc, nil
}

func (bc *BoardAeroClient) CheckConnection() error {
	if bc.aeroClient != nil && bc.aeroClient.IsConnected() {
		return nil
	}

	if bc.isConnecting {
		return errors.New("aero client already connecting")
	}

	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	bc.isConnecting = true
	defer func() {
		bc.isConnecting = false
	}()

	log.Debugf("trying to connect to aerospike server %s:%d [namespace:%s, set:%s] ...",
		bc.host, bc.port, bc.namespace, bc.set)

	aeroClient, err := as.NewClient(bc.host, bc.port)
	if err != nil {
		return fmt.Errorf("failed to create aero client / connect to aero: %w", err)
	}

	bc.aeroClient = aeroClient
	log.Debug("aero client successfully connected")

	return nil
}

func (bc *BoardAeroClient) GetMessageIdCounter() (int, error) {
	if err := bc.CheckConnection(); err != nil {
		return -1, err
	}

	key, err := as.NewKey(bc.namespace, bc.metaDataSet, "message-id-counter")
	if err != nil {
		return -1, err
	}

	record, err := bc.aeroClient.Get(nil, key)
	if err != nil {
		return -1, err
	}

	counterRaw, ok := record.Bins["idCounter"]
	if !ok {
		return -1, errors.New("id counter not existing")
	}

	counter, ok := counterRaw.(int)
	if !ok {
		return -1, errors.New("id counter not an integer")
	}

	return counter, nil
}

func (bc *BoardAeroClient) IncrementMessageIdCounter(increment int) (int, error) {
	if err := bc.CheckConnection(); err != nil {
		return -1, err
	}

	key, err := as.NewKey(bc.namespace, bc.metaDataSet, "message-id-counter")
	if err != nil {
		return -1, err
	}

	counterBin := as.NewBin("idCounter", increment)
	record, err := bc.aeroClient.Operate(nil, key, as.AddOp(counterBin), as.GetOp())
	if err != nil {
		return -1, fmt.Errorf("failed to call aero operate: %w", err)
	}

	counterRaw, ok := record.Bins["idCounter"]
	if !ok {
		return -1, errors.New("id counter not existing")
	}

	counter, ok := counterRaw.(int)
	if !ok {
		return -1, errors.New("id counter not an integer")
	}

	return counter, nil
}

func (bc *BoardAeroClient) Put(key string, binMap AeroBinMap) error {
	if err := bc.CheckConnection(); err != nil {
		return err
	}

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
	if err := bc.CheckConnection(); err != nil {
		return false, err
	}

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
	if err := bc.CheckConnection(); err != nil {
		return nil, err
	}

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

	return RecordSet2AeroBinMaps(recordSet)
}

func (bc *BoardAeroClient) ScanAll() ([]AeroBinMap, error) {
	if err := bc.CheckConnection(); err != nil {
		return nil, err
	}

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = true

	recordSet, err := bc.aeroClient.ScanAll(spolicy, bc.namespace, bc.set)
	if err != nil {
		return nil, err
	}

	return RecordSet2AeroBinMaps(recordSet)
}

func (bc *BoardAeroClient) CountAll() (int, error) {
	if err := bc.CheckConnection(); err != nil {
		return -1, err
	}

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = false

	recs, err := bc.aeroClient.ScanAll(spolicy, bc.namespace, bc.set)
	if err != nil {
		return -1, err
	}

	count := 0
	for range recs.Results() {
		count++
	}

	return count, nil
}

func (bc *BoardAeroClient) Close() {
	if bc.aeroClient != nil && bc.aeroClient.IsConnected() {
		bc.aeroClient.Close()
	}
}
