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
	allMessagesCacheKey  = "all-messages"
	BoardMessagesSetName = "messages"
)

type Board struct {
	// aerospike data model (namespace, set, record, bin, ...) infos:
	// https://aerospike.com/docs/architecture/data-model.html
	aeroClient *as.Client

	aeroNamespace string
	messagesSet   string
	messagesCount int
	cache         *ristretto.Cache

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
		messagesSet:   BoardMessagesSetName,
		cache:         cache,
	}

	messagesCount, err := b.MessagesCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get all messages count: %w", err)
	}
	b.messagesCount = messagesCount

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
	// not really sure, all messages should be really cached
	b.CacheBoardMessages(allMessagesCacheKey, allMessages)
}

func (b *Board) CacheBoardMessages(cacheKey string, messages []*BoardMessage) {
	if !b.cache.Set(cacheKey, messages, int64(len(messages)*3)) {
		log.Errorf("failed to set cache for [%s]... for some reason", cacheKey)
	} else {
		log.Debugf("board messages cache set for [%s]", cacheKey)
	}
}

func (b *Board) MessagesPageCacheKey(page, size int) string {
	return fmt.Sprintf("messages::%d::%d", page, size)
}

func (b *Board) InvalidateCaches() {
	log.Tracef("invalidating cache")
	b.cache.Clear()
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

	// TODO: check write/read policies, and whether I need them
	writePolicy := as.NewWritePolicy(0, 0)
	err = b.aeroClient.Put(writePolicy, key, bins)
	if err != nil {
		return err
	}

	// omg, fix this laziness
	b.InvalidateCaches()

	b.messagesCount++

	return nil
}

func (b *Board) GetMessagesPage(page, size int) ([]*BoardMessage, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	log.Tracef("getting messages page %d, size %d", page, size)

	if size >= b.messagesCount {
		return b.AllMessagesCache(false)
	}

	cacheKey := b.MessagesPageCacheKey(page, size)
	if cachedMessages, found := b.cache.Get(cacheKey); found {
		if messages, ok := cachedMessages.([]*BoardMessage); ok {
			log.Tracef("%d messages found for page %d and size %d", len(messages), page, size)
			return messages, nil
		}
		return nil, fmt.Errorf("failed to convert messages cache")
	}
	// cache miss here, will get messages from aero and cache them

	pages := (b.messagesCount / size) + 1

	var from, to int64
	if page >= pages {
		from = int64(b.messagesCount - size)
		to = int64(b.messagesCount)
	} else {
		from = int64((page - 1) * size)
		to = from + int64(size-1)
	}

	rangeFilterStt := &as.Statement{
		Namespace: b.aeroNamespace,
		SetName:   b.messagesSet,
		IndexName: "id",
		Filter:    as.NewRangeFilter("id", from, to),
	}

	recordSet, err := b.aeroClient.Query(nil, rangeFilterStt)
	if err != nil {
		return nil, fmt.Errorf("failed to query aero for range filter set: %w", err)
	}

	var messages []*BoardMessage
	for rec := range recordSet.Results() {
		if rec.Err != nil {
			log.Errorf("get messages page, record error: %s", rec.Err)
			continue
		}
		m := MessageFromBins(rec.Record.Bins)
		messages = append(messages, &m)
	}

	log.Tracef("received %d messages from aerospike", len(messages))

	b.CacheBoardMessages(cacheKey, messages)

	return messages, nil
}

func (b *Board) GetMessagesWithRange(from, to int64) ([]*BoardMessage, error) {
	log.Tracef("getting messages range from %d to %d", from, to)

	rangeFilterStt := &as.Statement{
		Namespace: b.aeroNamespace,
		SetName:   b.messagesSet,
		IndexName: "id",
		Filter:    as.NewRangeFilter("id", from, to),
	}

	recordSet, err := b.aeroClient.Query(nil, rangeFilterStt)
	if err != nil {
		return nil, fmt.Errorf("failed to query aero for range filter set: %w", err)
	}

	var messages []*BoardMessage
	for rec := range recordSet.Results() {
		if rec.Err != nil {
			log.Errorf("get messages range, record error: %s", rec.Err)
			continue
		}
		m := MessageFromBins(rec.Record.Bins)
		messages = append(messages, &m)
	}

	log.Tracef("received %d messages from aerospike", len(messages))

	return messages, nil
}

func (b *Board) AllMessagesCache(sortByTimestamp bool) ([]*BoardMessage, error) {
	if allMessagesCached, found := b.cache.Get(allMessagesCacheKey); found {
		if allMessages, ok := allMessagesCached.([]*BoardMessage); ok {
			log.Tracef("all %d messages found in cache", len(allMessages))
			return allMessages, nil
		}
		return nil, fmt.Errorf("failed to convert all messages cache")
	}

	log.Errorf("failed to get all messages cache, will get them from aerospike")
	allMessages, err := b.AllMessages(sortByTimestamp)
	if err != nil {
		return nil, err
	}

	b.CacheBoardMessages(allMessagesCacheKey, allMessages)

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
