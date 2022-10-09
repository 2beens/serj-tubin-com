package board

import (
	"strconv"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/cache"
	"github.com/2beens/serjtubincom/internal/testinternals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBoard(t *testing.T) {
	board, err := NewClient(nil, cache.NewBoardTestCache())
	assert.Equal(t, aerospike.ErrAeroClientNil, err)
	assert.Nil(t, board)

	aeroTestClient := aerospike.NewBoardAeroTestClient()
	board, err = NewClient(aeroTestClient, cache.NewBoardTestCache())
	require.NoError(t, err)
	require.NotNil(t, board)
}

func TestBoard_AllMessagesCache(t *testing.T) {
	internals := testinternals.NewTestingInternals()
	board := internals.BoardClient

	// cache empty at the beginning
	require.Equal(t, 0, internals.BoardCache.ElementsCount())

	messages, err := board.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.InitialBoardMessages))

	// 1 cache entry - all messages (in that entry are all 3 messages)
	require.Equal(t, 1, internals.BoardCache.ElementsCount())
	messagesFromCacheRaw, found := internals.BoardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok := messagesFromCacheRaw.([]*Message)
	require.True(t, ok)
	require.Len(t, messagesFromCache, len(internals.InitialBoardMessages))

	funcCallsLog := internals.BoardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 3)
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, cache.FuncSet, funcCallsLog[1])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[2])

	internals.BoardCache.ClearFunctionCallsLog()

	// called again - should get it from cache right away
	messages, err = board.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.InitialBoardMessages))

	require.Equal(t, 1, internals.BoardCache.ElementsCount())
	messagesFromCacheRaw, found = internals.BoardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok = messagesFromCacheRaw.([]*Message)
	require.True(t, ok)
	assert.Len(t, messagesFromCache, len(internals.InitialBoardMessages))

	funcCallsLog = internals.BoardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 2)
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[0])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[1])
}

func TestBoard_AllMessages(t *testing.T) {
	internals := testinternals.NewTestingInternals()

	messages, err := internals.BoardClient.AllMessages(false)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.InitialBoardMessages))

	messages, err = internals.BoardClient.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(internals.InitialBoardMessages))
	// sorted by timestamp
	assert.Equal(t, 3, messages[0].ID)
	assert.Equal(t, 2, messages[1].ID)
	assert.Equal(t, 0, messages[2].ID)
	assert.Equal(t, 4, messages[3].ID)
	assert.Equal(t, 1, messages[4].ID)
}

func TestBoard_DeleteMessage(t *testing.T) {
	internals := testinternals.NewTestingInternals()
	board := internals.BoardClient

	// non existent message
	removed, err := board.DeleteMessage("100")
	require.NoError(t, err)
	assert.False(t, removed)

	// existent message
	messagesCount, err := board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.InitialBoardMessages), messagesCount)

	removed, err = board.DeleteMessage("1")
	require.NoError(t, err)
	assert.True(t, removed)

	messagesCount, err = board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.InitialBoardMessages)-1, messagesCount)
}

func TestBoard_SetAllMessagesCacheFromAero(t *testing.T) {
	internals := testinternals.NewTestingInternals()

	// cache empty
	require.Equal(t, 0, internals.BoardCache.ElementsCount())

	assert.NoError(t, internals.BoardClient.SetAllMessagesCacheFromAero())

	// cache filled
	require.Equal(t, 1, internals.BoardCache.ElementsCount())
	allMessagesFromCache, found := internals.BoardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, allMessagesFromCache)
	allMessages, ok := allMessagesFromCache.([]*Message)
	require.True(t, ok)
	assert.Len(t, allMessages, len(internals.InitialBoardMessages))
}

func TestBoard_CacheBoardMessages(t *testing.T) {
	internals := testinternals.NewTestingInternals()

	// cache empty
	require.Equal(t, 0, internals.BoardCache.ElementsCount())

	messages := []*Message{
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

	internals.BoardClient.CacheBoardMessages("messages", messages)

	// cache filled
	require.Equal(t, 1, internals.BoardCache.ElementsCount())

	allMessagesFromCache, found := internals.BoardCache.Get("messages")
	require.True(t, found)
	require.NotNil(t, allMessagesFromCache)
	allMessages, ok := allMessagesFromCache.([]*Message)
	require.True(t, ok)
	assert.Len(t, allMessages, 2)
}

func TestBoard_InvalidateCaches(t *testing.T) {
	internals := testinternals.NewTestingInternals()

	// cache empty
	require.Equal(t, 0, internals.BoardCache.ElementsCount())

	internals.BoardClient.CacheBoardMessages("messages", []*Message{
		{
			ID:        0,
			Author:    "a0",
			Timestamp: time.Now().Unix(),
			Message:   "m0",
		},
	})

	// cache filled
	require.Equal(t, 1, internals.BoardCache.ElementsCount())

	internals.BoardClient.InvalidateCaches()
	// cache empty
	require.Equal(t, 0, internals.BoardCache.ElementsCount())
}

func TestBoard_StoreMessage(t *testing.T) {
	internals := testinternals.NewTestingInternals()
	board := internals.BoardClient

	messagesCount, err := board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.InitialBoardMessages), messagesCount)

	now := time.Now()
	m1 := Message{
		ID:        len(internals.InitialBoardMessages),
		Author:    "ana",
		Timestamp: now.Unix(),
		Message:   "lixo",
	}
	newId1, err := board.NewMessage(m1)
	require.NoError(t, err)
	assert.Equal(t, m1.ID, newId1)

	m2 := Message{
		ID:        len(internals.InitialBoardMessages) + 1,
		Author:    "serj",
		Timestamp: now.Add(-time.Hour).Unix(),
		Message:   "a message",
	}
	newId2, err := board.NewMessage(m2)
	require.NoError(t, err)
	assert.Equal(t, m2.ID, newId2)

	messagesCount, err = board.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(internals.InitialBoardMessages)+2, messagesCount)

	allMessages, err := board.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, allMessages, len(internals.InitialBoardMessages)+2)

	m1binMap := internals.AeroTestClient.AeroBinMaps[strconv.Itoa(m1.ID)]
	require.NotNil(t, m1binMap)
	assert.Equal(t, m1.Message, m1binMap["message"])
	m2binMap := internals.AeroTestClient.AeroBinMaps[strconv.Itoa(m2.ID)]
	require.NotNil(t, m2binMap)
	assert.Equal(t, m2.Message, m2binMap["message"])
}

func TestBoard_GetMessagesWithRange(t *testing.T) {
	internals := testinternals.NewTestingInternals()

	// cache empty at the beginning
	require.Equal(t, 0, internals.BoardCache.ElementsCount())

	messages, err := internals.BoardClient.GetMessagesWithRange(1, 3)
	require.NoError(t, err)

	// cache empty after - GetMessagesWithRange does not cache atm
	require.Equal(t, 0, internals.BoardCache.ElementsCount())
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
	internals := testinternals.NewTestingInternals()
	board := internals.BoardClient

	// cache empty at the beginning
	require.Equal(t, 0, internals.BoardCache.ElementsCount())

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
	funcCallsLog := internals.BoardCache.FunctionCallsLog
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
	funcCallsLog = internals.BoardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 7)
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, cache.FuncSet, funcCallsLog[1])
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[2])
	assert.Equal(t, cache.FuncSet, funcCallsLog[3])
	assert.Equal(t, cache.FuncGetMiss, funcCallsLog[4])
	assert.Equal(t, cache.FuncSet, funcCallsLog[5])
	assert.Equal(t, cache.FuncGetHit, funcCallsLog[6])
}
