package internal

import (
	"fmt"
	"sort"
	"sync"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/dgraph-io/ristretto"
	log "github.com/sirupsen/logrus"
)

// TODO: unit tests <3

const (
	AllMessagesCacheKey = "all-messages"
)

type Board struct {
	// TODO: aerospike data model (namespace, set, record, bin, ...) infos:
	// https://aerospike.com/docs/architecture/data-model.html
	aeroClient    *as.Client
	aeroNamespace string
	messagesSet   string
	cache         *ristretto.Cache
	messagesCount int

	mutex sync.RWMutex
}

func NewBoard(aeroHost string, aeroPort int, aeroNamespace string) (*Board, error) {
	log.Debugf("connecting to aerospike server %s:%d ...", aeroHost, aeroPort)

	aeroClient, err := as.NewClient(aeroHost, aeroPort)
	if err != nil {
		return nil, err
	}

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M)
		MaxCost:     1 << 28, // maximum cost of cache (~268M)
		BufferItems: 64,      // number of keys per Get buffer
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %s", err)
	}

	b := &Board{
		aeroClient:    aeroClient,
		aeroNamespace: aeroNamespace,
		messagesSet:   "messages",
		cache:         cache,
	}

	messagesCount, err := b.MessagesCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get all messages count: %w", err)
	}
	b.messagesCount = messagesCount

	// https://www.aerospike.com/docs/client/go/usage/query/sindex.html
	// TODO:
	// check index created
	//		create if not
	// actually: create index once using AQL:
	//		https://www.aerospike.com/docs/operations/manage/indexes/index.html
	//aeroClient.CreateIndex()

	// what I need for timestamps:
	//	https://www.aerospike.com/docs/client/java/examples/application/queries.html#retrieve-all-user-records-using-tweetcount-

	// TODO: testings
	//f := as.Filter
	//as.ListGetByIndexRangeCountOp()
	//_ = f

	//aeroClient.CreateIndex()

	go b.SetAllMessagesCacheFromAero()

	return b, nil
}

func (b *Board) SetAllMessagesCacheFromAero() {
	allMessages, err := b.AllMessages(true)
	if err != nil {
		log.Errorf("failed to prepare visitor board cache: %s", err)
		return
	}
	// TODO: this is a super lazy way to cache messages
	b.SetAllMessagesCache(allMessages)
}

func (b *Board) SetAllMessagesCache(allMessages []*BoardMessage) {
	if !b.cache.Set(AllMessagesCacheKey, allMessages, int64(len(allMessages)*3)) {
		log.Errorf("failed to set all messages to cache... for some reason")
	} else {
		log.Debug("all board messages cache set")
	}
}

func (b *Board) Close() {
	if b != nil && b.aeroClient != nil {
		b.aeroClient.Close()
	}
}

func (b *Board) StoreMessage(message BoardMessage) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.aeroClient == nil {
		return fmt.Errorf("aero client is nil")
	}

	key, err := as.NewKey(b.aeroNamespace, b.messagesSet, b.messagesCount)
	if err != nil {
		return err
	}

	log.Debugf("saving message %d: %+v: %s - %s", b.messagesCount, message.Timestamp, message.Author, message.Message)

	bins := as.BinMap{
		"id":        b.messagesCount,
		"author":    message.Author,
		"timestamp": message.Timestamp,
		"message":   message.Message,
	}

	writePolicy := as.NewWritePolicy(0, 0)
	err = b.aeroClient.Put(writePolicy, key, bins)
	if err != nil {
		return err
	}

	// omg, fix this laziness
	b.cache.Del(AllMessagesCacheKey)

	b.messagesCount++

	return nil
}

// TODO: check if sorting by timestamp even works
func (b *Board) AllMessagesCache(sortByTimestamp bool) ([]*BoardMessage, error) {
	allMessagesRaw, found := b.cache.Get(AllMessagesCacheKey)
	if !found {
		log.Errorf("failed to get all messages cache, will get them from aerospike")
		allMessages, err := b.AllMessages(sortByTimestamp)
		if err != nil {
			return nil, err
		}
		b.SetAllMessagesCache(allMessages)
		return allMessages, nil
	}

	allMessages, ok := allMessagesRaw.([]*BoardMessage)
	if !ok {
		return nil, fmt.Errorf("failed to convert all messages cache, will get them from aerospike")
	}

	log.Tracef("all %d messages found in cache", len(allMessages))

	return allMessages, nil
}

func (b *Board) AllMessages(sortByTimestamp bool) ([]*BoardMessage, error) {
	if b.aeroClient == nil {
		return nil, fmt.Errorf("aero client is nil")
	}

	log.Tracef("getting all messages from Aerospike, namespace: %s, set: %s", b.aeroNamespace, b.messagesSet)

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = true

	recordSet, err := b.aeroClient.ScanAll(spolicy, b.aeroNamespace, b.messagesSet)
	if err != nil {
		return nil, err
	}

	// TODO: maybe try getting all keys first, and initiate a batch Read ?

	var messages []*BoardMessage
	for rec := range recordSet.Results() {
		if rec.Err != nil {
			log.Errorf("get all messages, record error: %s", rec.Err)
			continue
		}
		m := MessageFromBins(rec.Record.Bins)
		messages = append(messages, &m)
	}

	log.Tracef("received %d messages from aerospike", len(messages))

	if sortByTimestamp {
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Timestamp < messages[j].Timestamp
		})
	}

	return messages, nil
}

func (b *Board) MessagesCount() (int, error) {
	if b.aeroClient == nil {
		return -1, fmt.Errorf("aero client is nil")
	}

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = false

	recs, err := b.aeroClient.ScanAll(spolicy, b.aeroNamespace, b.messagesSet)
	if err != nil {
		return -1, err
	}

	count := 0
	for _ = range recs.Results() {
		count++
	}

	return count, nil
}
