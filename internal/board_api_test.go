package internal

import (
	"strconv"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBoard(t *testing.T) {
	board, err := NewBoardApi(nil, cache.NewBoardTestCache())
	assert.Equal(t, aerospike.ErrAeroClientNil, err)
	assert.Nil(t, board)

	aeroTestClient := aerospike.NewBoardAeroTestClient()
	board, err = NewBoardApi(aeroTestClient, cache.NewBoardTestCache())
	require.NoError(t, err)
	require.NotNil(t, board)
}

func TestBoard_AllMessagesCache(t *testing.T) {
	internals := newTestingInternals()
	board := internals.boardApi

	// cache empty at the beginning
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	messages, err := board.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.initialBoardMessages))

	// 1 cache entry - all messages (in that entry are all 3 messages)
	require.Equal(t, 1, internals.boardCache.ElementsCount())
	messagesFromCacheRaw, found := internals.boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok := messagesFromCacheRaw.([]*BoardMessage)
	require.True(t, ok)
	require.Len(t, messagesFromCache, len(internals.initialBoardMessages))

	funcCallsLog := internals.boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 3)
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, cache.FuncSet, funcCallsLog[1])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[2])

	internals.boardCache.ClearFunctionCallsLog()

	// called again - should get it from cache right away
	messages, err = board.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.initialBoardMessages))

	require.Equal(t, 1, internals.boardCache.ElementsCount())
	messagesFromCacheRaw, found = internals.boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok = messagesFromCacheRaw.([]*BoardMessage)
	require.True(t, ok)
	assert.Len(t, messagesFromCache, len(internals.initialBoardMessages))

	funcCallsLog = internals.boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 2)
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[0])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[1])
}

func TestBoard_AllMessages(t *testing.T) {
	internals := newTestingInternals()

	messages, err := internals.boardApi.AllMessages(false)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.initialBoardMessages))

	messages, err = internals.boardApi.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.initialBoardMessages))
	// sorted by timestamp
	assert.Equal(t, 4, messages[0].ID)
	assert.Equal(t, 3, messages[1].ID)
	assert.Equal(t, 1, messages[2].ID)
	assert.Equal(t, 5, messages[3].ID)
	assert.Equal(t, 2, messages[4].ID)

}

func TestBoard_DeleteMessage(t *testing.T) {
	internals := newTestingInternals()
	board := internals.boardApi

	// non existent message
	removed, err := board.DeleteMessage("100")
	require.NoError(t, err)
	assert.False(t, removed)

	// existent message
	messagesCount, err := board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.initialBoardMessages), messagesCount)

	removed, err = board.DeleteMessage("1")
	require.NoError(t, err)
	assert.True(t, removed)

	messagesCount, err = board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.initialBoardMessages)-1, messagesCount)
}

func TestBoard_CacheBoardMessages(t *testing.T) {
	internals := newTestingInternals()

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

	internals.boardApi.CacheBoardMessages("messages", messages)

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
	internals := newTestingInternals()

	// cache empty
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	internals.boardApi.CacheBoardMessages("messages", []*BoardMessage{
		{
			ID:        0,
			Author:    "a0",
			Timestamp: time.Now().Unix(),
			Message:   "m0",
		},
	})

	// cache filled
	require.Equal(t, 1, internals.boardCache.ElementsCount())

	internals.boardApi.InvalidateCaches()
	// cache empty
	require.Equal(t, 0, internals.boardCache.ElementsCount())
}

func TestBoard_StoreMessage(t *testing.T) {
	internals := newTestingInternals()
	board := internals.boardApi

	messagesCount, err := board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.initialBoardMessages), messagesCount)

	alreadyAddedMessagesCount := len(internals.initialBoardMessages)

	now := time.Now()
	m1 := BoardMessage{
		Author:    "ana",
		Timestamp: now.Unix(),
		Message:   "lixo",
	}
	newId1, err := board.StoreMessage(m1)
	require.NoError(t, err)
	assert.Equal(t, 1+alreadyAddedMessagesCount, newId1)
	m1.ID = newId1

	m2 := BoardMessage{
		Author:    "serj",
		Timestamp: now.Add(-time.Hour).Unix(),
		Message:   "a message",
	}
	newId2, err := board.StoreMessage(m2)
	require.NoError(t, err)
	assert.Equal(t, 2+alreadyAddedMessagesCount, newId2)
	m2.ID = newId2

	messagesCount, err = board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.initialBoardMessages)+2, messagesCount)

	allMessages, err := board.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, allMessages, len(internals.initialBoardMessages)+2)

	m1binMap := internals.aeroTestClient.AeroBinMaps[strconv.Itoa(m1.ID)]
	require.NotNil(t, m1binMap)
	assert.Equal(t, m1.Message, m1binMap["message"])
	m2binMap := internals.aeroTestClient.AeroBinMaps[strconv.Itoa(m2.ID)]
	require.NotNil(t, m2binMap)
	assert.Equal(t, m2.Message, m2binMap["message"])
}

func TestBoard_GetMessagesWithRange(t *testing.T) {
	internals := newTestingInternals()

	// cache empty at the beginning
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	messages, err := internals.boardApi.GetMessagesWithRange(2, 4)
	require.NoError(t, err)

	// cache empty after - GetMessagesWithRange does not cache atm
	require.Equal(t, 0, internals.boardCache.ElementsCount())
	require.Len(t, messages, 3)

	// order not guaranteed
	var found1, found2, found3 bool
	for i := range messages {
		if messages[i].Message == "test message gragra" {
			found1 = true
		}
		if messages[i].Message == "test message aaaaa" {
			found2 = true
		}
		if messages[i].Message == "drago's test message aaaaa sve" {
			found3 = true
		}
	}

	assert.True(t, found1)
	assert.True(t, found2)
	assert.True(t, found3)
}

func TestBoard_GetMessagesPage(t *testing.T) {
	internals := newTestingInternals()
	board := internals.boardApi

	// cache empty at the beginning
	require.Equal(t, 0, internals.boardCache.ElementsCount())

	messages, err := board.GetMessagesPage(2, 2)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	// order not guaranteed
	var found1, found2 bool
	for i := range messages {
		if messages[i].Message == "test message aaaaa" {
			found1 = true
		}
		if messages[i].Message == "drago's test message aaaaa sve" {
			found2 = true
		}
	}
	assert.True(t, found1)
	assert.True(t, found2)

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

	found1 = false
	found2 = false
	for i := range messages {
		if messages[i].Message == "test message aaaaa" {
			found1 = true
		}
		if messages[i].Message == "drago's test message aaaaa sve" {
			found2 = true
		}
	}
	assert.True(t, found1)
	assert.True(t, found2)

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
