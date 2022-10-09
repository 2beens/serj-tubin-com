package board

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

type Client struct {
	aeroClient aerospike.Client
	cache      cache.Cache
	mutex      sync.RWMutex
}

func NewClient(aeroClient aerospike.Client, cache cache.Cache) (*Client, error) {
	if aeroClient == nil {
		return nil, aerospike.ErrAeroClientNil
	}

	b := &Client{
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

func (c *Client) SetAllMessagesCacheFromAero() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	allMessages, err := c.AllMessages(true)
	if err != nil {
		return err
	}

	// TODO: this is a super lazy way to cache messages
	// not really sure, all messages should be really cached
	c.CacheBoardMessages(AllMessagesCacheKey, allMessages)

	return nil
}

func (c *Client) CacheBoardMessages(cacheKey string, messages []*Message) {
	if !c.cache.Set(cacheKey, messages, int64(len(messages)*3)) {
		log.Errorf("failed to set cache for [%s]... for some reason", cacheKey)
	} else {
		log.Debugf("board messages cache set for [%s]", cacheKey)
	}
}

func (c *Client) MessagesPageCacheKey(page, size int) string {
	return fmt.Sprintf("messages::%d::%d", page, size)
}

func (c *Client) InvalidateCaches() {
	log.Tracef("invalidating cache")
	c.cache.Clear()
}

func (c *Client) Close() {
	if c != nil && c.aeroClient != nil {
		c.aeroClient.Close()
	}
}

func (c *Client) NewMessage(message Message) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	newMessageId, err := c.aeroClient.IncrementMessageIdCounter(1)
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
	if err := c.aeroClient.Put(messageKey, bins); err != nil {
		return -1, fmt.Errorf("failed to do aero put: %w", err)
	}

	// omg, fix this laziness
	c.InvalidateCaches()

	return newMessageId, nil
}

func (c *Client) DeleteMessage(messageId string) (bool, error) {
	log.Tracef("board - about to delete message: %s", messageId)
	c.InvalidateCaches()
	return c.aeroClient.Delete(messageId)
}

func (c *Client) GetMessagesPage(page, size int) ([]*Message, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Tracef("getting messages page %d, size %d", page, size)

	totalMessagesCount, err := c.MessagesCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get all messages count: %w", err)
	}

	if size >= totalMessagesCount {
		return c.AllMessagesCache(false)
	}

	cacheKey := c.MessagesPageCacheKey(page, size)
	if cachedMessages, found := c.cache.Get(cacheKey); found {
		if messages, ok := cachedMessages.([]*Message); ok {
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

	messagesBins, err := c.aeroClient.QueryByRange("id", from, to)
	if err != nil {
		return nil, fmt.Errorf("get messages page, failed to query aero spike for messages: %w", err)
	}

	var messages []*Message
	for _, mBin := range messagesBins {
		m := MessageFromBins(mBin)
		messages = append(messages, &m)
	}

	log.Tracef("received %d messages from aerospike", len(messages))

	c.CacheBoardMessages(cacheKey, messages)

	return messages, nil
}

func (c *Client) GetMessagesWithRange(from, to int64) ([]*Message, error) {
	log.Tracef("getting messages range from %d to %d", from, to)

	messagesBins, err := c.aeroClient.QueryByRange("id", from, to)
	if err != nil {
		return nil, fmt.Errorf("get messages with range, failed to query aero spike for messages: %w", err)
	}

	var messages []*Message
	for _, mBin := range messagesBins {
		m := MessageFromBins(mBin)
		messages = append(messages, &m)
	}

	log.Tracef("received %d messages from aerospike", len(messages))

	return messages, nil
}

func (c *Client) AllMessagesCache(sortByTimestamp bool) ([]*Message, error) {
	if allMessagesCached, found := c.cache.Get(AllMessagesCacheKey); found {
		if allMessages, ok := allMessagesCached.([]*Message); ok {
			log.Tracef("all %d messages found in cache", len(allMessages))
			return allMessages, nil
		}
		return nil, fmt.Errorf("failed to convert all messages cache")
	}

	log.Errorf("failed to get all messages cache, will get them from aerospike")
	allMessages, err := c.AllMessages(sortByTimestamp)
	if err != nil {
		return nil, err
	}

	c.CacheBoardMessages(AllMessagesCacheKey, allMessages)

	return allMessages, nil
}

func (c *Client) AllMessages(sortByTimestamp bool) ([]*Message, error) {
	log.Tracef("getting all messages from Aerospike")

	messagesBins, err := c.aeroClient.ScanAll()
	if err != nil {
		return nil, fmt.Errorf("get all messages, failed to query aero spike for messages: %w", err)
	}

	var messages []*Message
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

func (c *Client) MessagesCount() (int, error) {
	return c.aeroClient.CountAll()
}
