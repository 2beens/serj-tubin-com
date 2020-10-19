package internal

import (
	"log"
	"strconv"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestingInternals struct {
	aeroTestClient *aerospike.BoardAeroTestClient
	boardCache     *cache.BoardTestCache
	board          *Board
}

var initialMessagesCount int

func newTestingInternals() (*TestingInternals, *Board) {
	aeroClient := aerospike.NewBoardAeroTestClient()
	boardCache := cache.NewBoardTestCache()

	board, err := NewBoard(aeroClient, boardCache, "aero-test")
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	if err := aeroClient.Put("0", aerospike.AeroBinMap{
		"id":        0,
		"author":    "serj",
		"message":   "test message blabla",
		"timestamp": now.Add(-time.Hour).Unix(),
	}); err != nil {
		panic(err)
	}
	if err := aeroClient.Put("1", aerospike.AeroBinMap{
		"id":        1,
		"author":    "serj",
		"message":   "test message gragra",
		"timestamp": now.Unix(),
	}); err != nil {
		panic(err)
	}
	if err := aeroClient.Put("2", aerospike.AeroBinMap{
		"id":        2,
		"author":    "ana",
		"message":   "test message aaaaa",
		"timestamp": now.Add(-2 * time.Hour).Unix(),
	}); err != nil {
		panic(err)
	}
	if err := aeroClient.Put("3", aerospike.AeroBinMap{
		"id":        3,
		"author":    "drago",
		"message":   "drago's test message aaaaa sve",
		"timestamp": now.Add(-5 * 24 * time.Hour).Unix(),
	}); err != nil {
		panic(err)
	}
	if err := aeroClient.Put("4", aerospike.AeroBinMap{
		"id":        4,
		"author":    "rodjak nenad",
		"message":   "ja se mislim sta'e bilo",
		"timestamp": now.Add(-2 * time.Minute).Unix(),
	}); err != nil {
		panic(err)
	}

	initialMessagesCount = len(aeroClient.AeroBinMaps)
	board.messagesCount = initialMessagesCount

	return &TestingInternals{
		aeroTestClient: aeroClient,
		boardCache:     boardCache,
		board:          board,
	}, board
}

func TestNewBoard(t *testing.T) {
	board, err := NewBoard(nil, cache.NewBoardTestCache(), "aero-test")
	assert.Equal(t, aerospike.ErrAeroClientNil, err)
	assert.Nil(t, board)

	aeroTestClient := aerospike.NewBoardAeroTestClient()
	board, err = NewBoard(aeroTestClient, cache.NewBoardTestCache(), "aero-test")
	require.NoError(t, err)
	require.NotNil(t, board)

	assert.Equal(t, nil, board.CheckAero())
}

func TestBoard_CheckAero(t *testing.T) {
	internals, board := newTestingInternals()
	internals.aeroTestClient.IsConnectedValue = false
	assert.Equal(t, aerospike.ErrAeroClientNotConnected, board.CheckAero())
}

func TestBoard_AllMessagesCache(t *testing.T) {
	internals, board := newTestingInternals()

	// cache empty at the beginning
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	messages, err := board.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, initialMessagesCount)

	// 1 cache entry - all messages (in that entry are all 3 messages)
	require.Equal(t, 1, internals.boardCache.ElementsCount())
	messagesFromCacheRaw, found := internals.boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok := messagesFromCacheRaw.([]*BoardMessage)
	require.True(t, ok)
	require.Len(t, messagesFromCache, initialMessagesCount)

	funcCallsLog := internals.boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 3)
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, cache.FuncSet, funcCallsLog[1])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[2])

	internals.boardCache.ClearFunctionCallsLog()

	// called again - should get it from cache right away
	messages, err = board.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, initialMessagesCount)

	require.Equal(t, 1, internals.boardCache.ElementsCount())
	messagesFromCacheRaw, found = internals.boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok = messagesFromCacheRaw.([]*BoardMessage)
	require.True(t, ok)
	assert.Len(t, messagesFromCache, initialMessagesCount)

	funcCallsLog = internals.boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 2)
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[0])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[1])
}

func TestBoard_AllMessages(t *testing.T) {
	_, board := newTestingInternals()

	messages, err := board.AllMessages(false)
	require.NoError(t, err)
	assert.Len(t, messages, initialMessagesCount)

	messages, err = board.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, messages, initialMessagesCount)
	// sorted by timestamp
	assert.Equal(t, 3, messages[0].ID)
	assert.Equal(t, 2, messages[1].ID)
	assert.Equal(t, 0, messages[2].ID)
	assert.Equal(t, 4, messages[3].ID)
	assert.Equal(t, 1, messages[4].ID)

}

func TestBoard_DeleteMessage(t *testing.T) {
	_, board := newTestingInternals()

	// non existent message
	removed, err := board.DeleteMessage("100")
	require.NoError(t, err)
	assert.False(t, removed)

	// existent message
	messagesCount, err := board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, initialMessagesCount, messagesCount)

	removed, err = board.DeleteMessage("1")
	require.NoError(t, err)
	assert.True(t, removed)

	messagesCount, err = board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, initialMessagesCount-1, messagesCount)
}

func TestBoard_SetAllMessagesCacheFromAero(t *testing.T) {
	internals, board := newTestingInternals()

	// cache empty
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	assert.NoError(t, board.SetAllMessagesCacheFromAero())

	// cache filled
	require.Equal(t, 1, internals.boardCache.ElementsCount())
	allMessagesFromCache, found := internals.boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, allMessagesFromCache)
	allMessages, ok := allMessagesFromCache.([]*BoardMessage)
	require.True(t, ok)
	assert.Len(t, allMessages, initialMessagesCount)
}

func TestBoard_CacheBoardMessages(t *testing.T) {
	internals, board := newTestingInternals()

	// cache empty
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	messages := []*BoardMessage{
		{
			ID:        0,
			Author:    "a0",
			Timestamp: time.Now().Unix(),
			Message:   "m0",
		},
		{
			ID:        1,
			Author:    "a1",
			Timestamp: time.Now().Unix(),
			Message:   "m1",
		},
	}

	board.CacheBoardMessages("messages", messages)

	// cache filled
	require.Equal(t, 1, internals.boardCache.ElementsCount())

	allMessagesFromCache, found := internals.boardCache.Get("messages")
	require.True(t, found)
	require.NotNil(t, allMessagesFromCache)
	allMessages, ok := allMessagesFromCache.([]*BoardMessage)
	require.True(t, ok)
	assert.Len(t, allMessages, 2)
}

func TestBoard_InvalidateCaches(t *testing.T) {
	internals, board := newTestingInternals()

	// cache empty
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	board.CacheBoardMessages("messages", []*BoardMessage{
		{
			ID:        0,
			Author:    "a0",
			Timestamp: time.Now().Unix(),
			Message:   "m0",
		},
	})

	// cache filled
	require.Equal(t, 1, internals.boardCache.ElementsCount())

	board.InvalidateCaches()
	// cache empty
	require.Equal(t, 0, internals.boardCache.ElementsCount())
}

func TestBoard_StoreMessage(t *testing.T) {
	internals, board := newTestingInternals()

	messagesCount, err := board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, initialMessagesCount, messagesCount)

	now := time.Now()
	m1 := BoardMessage{
		ID:        initialMessagesCount,
		Author:    "ana",
		Timestamp: now.Unix(),
		Message:   "lixo",
	}
	err = board.StoreMessage(m1)
	require.NoError(t, err)

	m2 := BoardMessage{
		ID:        initialMessagesCount + 1,
		Author:    "serj",
		Timestamp: now.Add(-time.Hour).Unix(),
		Message:   "a message",
	}
	err = board.StoreMessage(m2)
	require.NoError(t, err)

	messagesCount, err = board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, initialMessagesCount+2, messagesCount)

	allMessages, err := board.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, allMessages, initialMessagesCount+2)

	m1binMap := internals.aeroTestClient.AeroBinMaps[strconv.Itoa(m1.ID)]
	require.NotNil(t, m1binMap)
	assert.Equal(t, m1.Message, m1binMap["message"])
	m2binMap := internals.aeroTestClient.AeroBinMaps[strconv.Itoa(m2.ID)]
	require.NotNil(t, m2binMap)
	assert.Equal(t, m2.Message, m2binMap["message"])
}

func TestBoard_GetMessagesWithRange(t *testing.T) {
	_, board := newTestingInternals()

	messages, err := board.GetMessagesWithRange(1, 3)
	require.NoError(t, err)

	require.Len(t, messages, 3)
	assert.Equal(t, "test message gragra", messages[0].Message)
	assert.Equal(t, "test message aaaaa", messages[1].Message)
	assert.Equal(t, "drago's test message aaaaa sve", messages[2].Message)
}

func TestBoard_GetMessagesPage(t *testing.T) {
	internals, board := newTestingInternals()

	// cache empty at the beginning
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	messages, err := board.GetMessagesPage(2, 2)
	require.NoError(t, err)

	require.Len(t, messages, 2)
	assert.Equal(t, "test message aaaaa", messages[0].Message)
	assert.Equal(t, "drago's test message aaaaa sve", messages[1].Message)

	// cache calls check
	funcCallsLog := internals.boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 2)
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, cache.FuncSet, funcCallsLog[1])

	// size greater than total - get all messages
	messages, err = board.GetMessagesPage(2, 12)
	require.NoError(t, err)
	require.Len(t, messages, 5)

	// page greater than total pages - get last page
	messages, err = board.GetMessagesPage(10, 2)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	// first case again
	messages, err = board.GetMessagesPage(2, 2)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	assert.Equal(t, "test message aaaaa", messages[0].Message)
	assert.Equal(t, "drago's test message aaaaa sve", messages[1].Message)

	// cache calls check
	funcCallsLog = internals.boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 7)
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, cache.FuncSet, funcCallsLog[1])
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[2])
	assert.Equal(t, cache.FuncSet, funcCallsLog[3])
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[4])
	assert.Equal(t, cache.FuncSet, funcCallsLog[5])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[6])
}
