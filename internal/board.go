package internal

import (
	"fmt"
	"sort"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/dgraph-io/ristretto"
	log "github.com/sirupsen/logrus"
)

// TODO: unit tests <3
// TODO: maybe better would be to persist board with SQLite, and CitiesData in Aerospike

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
	if b.aeroClient == nil {
		return fmt.Errorf("aero client is nil")
	}

	key, err := as.NewKey(b.aeroNamespace, b.messagesSet, message.Timestamp)
	if err != nil {
		return err
	}

	log.Debugf("saving message: %+v: %s - %s", message.Timestamp, message.Author, message.Message)

	bins := as.BinMap{
		"author":    message.Author,
		"timestamp": message.Timestamp,
		"message":   message.Message,
	}

	writePolicy := as.NewWritePolicy(0, 0)
	err = b.aeroClient.Put(writePolicy, key, bins)
	if err != nil {
		return err
	}

	b.cache.Del(AllMessagesCacheKey)

	return nil
}

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
	return allMessages, nil
}

func (b *Board) AllMessages(sortByTimestamp bool) ([]*BoardMessage, error) {
	if b.aeroClient == nil {
		return nil, fmt.Errorf("aero client is nil")
	}

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = true

	recs, err := b.aeroClient.ScanAll(spolicy, b.aeroNamespace, b.messagesSet)
	if err != nil {
		return nil, err
	}

	// TODO: maybe try getting all keys first, and initiate a batch Read ?

	var messages []*BoardMessage
	for rec := range recs.Results() {
		if rec.Err != nil {
			log.Errorf("get all messages, record error: %s", err)
			continue
		}
		m := MessageFromBins(rec.Record.Bins)
		messages = append(messages, &m)
	}

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
