package internal

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/cache"
	log "github.com/sirupsen/logrus"
)

const (
	AllMessagesCacheKey = "all-messages"
)

type Board struct {
	aeroClient aerospike.Client
	cache      cache.Cache
	mutex      sync.RWMutex
}

func NewBoard(aeroClient aerospike.Client, cache cache.Cache) (*Board, error) {
	if aeroClient == nil {
		return nil, aerospike.ErrAeroClientNil
	}

	b := &Board{
		aeroClient: aeroClient,
		cache:      cache,
	}

	// wait a bit for aero to connect
	connTimeout := 2 * time.Second
	if err := aeroClient.WaitForReady(connTimeout); err != nil {
		// just log and try connecting at later point
		log.Errorf("aero client failed to connect after %s: %s", connTimeout, err)
		return b, nil
	}

	if messageIdCounter, err := aeroClient.GetMessageIdCounter(); err != nil {
		log.Errorf("failed to get message id counter: %s", err)
	} else {
		log.Debugf("visitor board, received message id counter: %d", messageIdCounter)
	}

	if messagesCount, err := b.MessagesCount(); err != nil {
		log.Errorf("failed to get all messages count: %s", err)
	} else if messagesCount > 0 {
		go func() {
			if err := b.SetAllMessagesCacheFromAero(); err != nil {
				log.Errorf("failed to set all messages cache from aero cache: %s", err)
			}
		}()
	}

	return b, nil
}

func (b *Board) SetAllMessagesCacheFromAero() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	allMessages, err := b.AllMessages(true)
	if err != nil {
		return err
	}

	// TODO: this is a super lazy way to cache messages
	// not really sure, all messages should be really cached
	b.CacheBoardMessages(AllMessagesCacheKey, allMessages)

	return nil
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

func (b *Board) NewMessage(message BoardMessage) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newMessageId, err := b.aeroClient.IncrementMessageIdCounter(1)
	if err != nil {
		return -1, fmt.Errorf("failed to get message id counter: %w", err)
	}

	bins := aerospike.AeroBinMap{
		"id":        newMessageId,
		"author":    message.Author,
		"timestamp": message.Timestamp,
		"message":   message.Message,
	}

	log.Debugf("saving message %d: %+v: %s - %s", newMessageId, message.Timestamp, message.Author, message.Message)

	messageKey := strconv.Itoa(newMessageId)
	if err := b.aeroClient.Put(messageKey, bins); err != nil {
		return -1, fmt.Errorf("failed to do aero put: %w", err)
	}

	// omg, fix this laziness
	b.InvalidateCaches()

	return newMessageId, nil
}

func (b *Board) DeleteMessage(messageId string) (bool, error) {
	log.Tracef("board - about to delete message: %s", messageId)
	b.InvalidateCaches()
	return b.aeroClient.Delete(messageId)
}

func (b *Board) GetMessagesPage(page, size int) ([]*BoardMessage, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	log.Tracef("getting messages page %d, size %d", page, size)

	totalMessagesCount, err := b.MessagesCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get all messages count: %w", err)
	}

	if size >= totalMessagesCount {
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

	pages := (totalMessagesCount / size) + 1

	var from, to int64
	if page >= pages {
		from = int64(totalMessagesCount - size)
		to = int64(totalMessagesCount)
	} else {
		from = int64((page - 1) * size)
		to = from + int64(size-1)
	}

	messagesBins, err := b.aeroClient.QueryByRange("id", from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query aero spike for messages: %w", err)
	}

	var messages []*BoardMessage
	for _, mBin := range messagesBins {
		m := MessageFromBins(mBin)
		messages = append(messages, &m)
	}

	log.Tracef("received %d messages from aerospike", len(messages))

	b.CacheBoardMessages(cacheKey, messages)

	return messages, nil
}

func (b *Board) GetMessagesWithRange(from, to int64) ([]*BoardMessage, error) {
	log.Tracef("getting messages range from %d to %d", from, to)

	messagesBins, err := b.aeroClient.QueryByRange("id", from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query aero spike for messages: %w", err)
	}

	var messages []*BoardMessage
	for _, mBin := range messagesBins {
		m := MessageFromBins(mBin)
		messages = append(messages, &m)
	}

	log.Tracef("received %d messages from aerospike", len(messages))

	return messages, nil
}

func (b *Board) AllMessagesCache(sortByTimestamp bool) ([]*BoardMessage, error) {
	if allMessagesCached, found := b.cache.Get(AllMessagesCacheKey); found {
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

	b.CacheBoardMessages(AllMessagesCacheKey, allMessages)

	return allMessages, nil
}

func (b *Board) AllMessages(sortByTimestamp bool) ([]*BoardMessage, error) {
	log.Tracef("getting all messages from Aerospike")

	messagesBins, err := b.aeroClient.ScanAll()
	if err != nil {
		return nil, fmt.Errorf("failed to query aero spike for messages: %w", err)
	}

	var messages []*BoardMessage
	for _, mBin := range messagesBins {
		m := MessageFromBins(mBin)
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
	return b.aeroClient.CountAll()
}
