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

type BoardApi struct {
	aeroClient aerospike.Client

	messagesIdCounter int
	cache             cache.Cache

	mutex sync.RWMutex
}

func NewBoardApi(aeroClient aerospike.Client, cache cache.Cache) (*BoardApi, error) {
	if aeroClient == nil {
		return nil, aerospike.ErrAeroClientNil
	}

	b := &BoardApi{
		aeroClient:        aeroClient,
		cache:             cache,
		messagesIdCounter: -1,
	}

	// wait a bit for aero to connect
	// (or a better way - change CheckConnection(...) in boardApi aero client so it signals when it gets connected)
	time.Sleep(time.Second)

	messagesIdCounter := b.GetMessagesIdCounter()
	log.Tracef("board api messages id counter: %d", messagesIdCounter)

	return b, nil
}

func (b *BoardApi) GetMessagesIdCounter() int {
	if b.messagesIdCounter >= 0 {
		return b.messagesIdCounter
	}

	allMessages, err := b.AllMessages(true)
	if err != nil {
		b.messagesIdCounter = -1
		log.Errorf("board api failed to get all messages from aero, for cache and counter: %s", err)
	} else {
		maxId := 0
		for _, m := range allMessages {
			if m.ID > maxId {
				maxId = m.ID
			}
		}
		b.messagesIdCounter = maxId

		// TODO: this is a super lazy way to cache messages
		// not really sure, all messages should be really cached
		b.CacheBoardMessages(AllMessagesCacheKey, allMessages)
	}

	log.Warnf("board api init, messages counter: %d", b.messagesIdCounter)

	return b.messagesIdCounter
}

func (b *BoardApi) CacheBoardMessages(cacheKey string, messages []*BoardMessage) {
	if !b.cache.Set(cacheKey, messages, int64(len(messages)*3)) {
		log.Errorf("failed to set cache for [%s]... for some reason", cacheKey)
	} else {
		log.Debugf("board api messages cache set for [%s]", cacheKey)
	}
}

func (b *BoardApi) MessagesPageCacheKey(page, size int) string {
	return fmt.Sprintf("messages::%d::%d", page, size)
}

func (b *BoardApi) InvalidateCaches() {
	log.Tracef("invalidating cache")
	b.cache.Clear()
}

func (b *BoardApi) Close() {
	if b != nil && b.aeroClient != nil {
		b.aeroClient.Close()
	}
}

func (b *BoardApi) StoreMessage(message BoardMessage) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newMessageId := b.GetMessagesIdCounter() + 1
	if newMessageId <= 0 {
		return -1, fmt.Errorf("invalid new message id: %d", newMessageId)
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

	// used as an ID for the next message
	b.messagesIdCounter++

	return newMessageId, nil
}

func (b *BoardApi) DeleteMessage(messageId string) (bool, error) {
	log.Tracef("board api - about to delete message: %s", messageId)
	b.InvalidateCaches()
	return b.aeroClient.Delete(messageId)
}

func (b *BoardApi) GetMessagesPage(page, size int) ([]*BoardMessage, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	log.Tracef("getting messages page %d, size %d", page, size)

	// FIXME/TODO: this wont work if many/some messages get deleted
	messagesCount := b.GetMessagesIdCounter()
	if size >= messagesCount {
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

	pages := (messagesCount / size) + 1

	var from, to int64
	if page >= pages {
		from = int64(messagesCount - size)
		to = int64(messagesCount)
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

func (b *BoardApi) GetMessagesWithRange(from, to int64) ([]*BoardMessage, error) {
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

func (b *BoardApi) AllMessagesCache(sortByTimestamp bool) ([]*BoardMessage, error) {
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

func (b *BoardApi) AllMessages(sortByTimestamp bool) ([]*BoardMessage, error) {
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

func (b *BoardApi) MessagesCount() (int, error) {
	return b.aeroClient.CountAll()
}
