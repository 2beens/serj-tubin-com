package board

import (
	"strconv"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestBoardClient() (*Client, *BoardTestCache, *aerospike.BoardAeroTestClient, map[int]*Message) {
	now := time.Now()
	initialBoardMessages := map[int]*Message{
		0: {
			ID:        0,
			Author:    "serj",
			Timestamp: now.Add(-time.Hour).Unix(),
			Message:   "test message blabla",
		},
		1: {
			ID:        1,
			Author:    "serj",
			Timestamp: now.Unix(),
			Message:   "test message gragra",
		},
		2: {
			ID:        2,
			Author:    "ana",
			Timestamp: now.Add(-2 * time.Hour).Unix(),
			Message:   "test message aaaaa",
		},
		3: {
			ID:        3,
			Author:    "drago",
			Timestamp: now.Add(-5 * 24 * time.Hour).Unix(),
			Message:   "drago's test message aaaaa sve",
		},
		4: {
			ID:        4,
			Author:    "rodjak nenad",
			Timestamp: now.Add(-2 * time.Minute).Unix(),
			Message:   "ja se mislim sta'e bilo",
		},
	}

	aeroClient := aerospike.NewBoardAeroTestClient()
	boardCache := NewBoardTestCache()
	boardClient, err := NewClient(aeroClient, boardCache)
	if err != nil {
		panic(err)
	}

	if _, err := boardClient.NewMessage(*initialBoardMessages[0]); err != nil {
		panic(err)
	}
	if _, err := boardClient.NewMessage(*initialBoardMessages[1]); err != nil {
		panic(err)
	}
	if _, err := boardClient.NewMessage(*initialBoardMessages[2]); err != nil {
		panic(err)
	}
	if _, err := boardClient.NewMessage(*initialBoardMessages[3]); err != nil {
		panic(err)
	}
	if _, err := boardClient.NewMessage(*initialBoardMessages[4]); err != nil {
		panic(err)
	}

	boardCache.ClearFunctionCallsLog()

	return boardClient, boardCache, aeroClient, initialBoardMessages
}

func TestNewBoard(t *testing.T) {
	board, err := NewClient(nil, NewBoardTestCache())
	assert.Equal(t, aerospike.ErrAeroClientNil, err)
	assert.Nil(t, board)

	aeroTestClient := aerospike.NewBoardAeroTestClient()
	board, err = NewClient(aeroTestClient, NewBoardTestCache())
	require.NoError(t, err)
	require.NotNil(t, board)
}

func TestBoard_AllMessagesCache(t *testing.T) {
	boardClient, boardCache, _, initialBoardMessages := getTestBoardClient()

	// cache empty at the beginning
	require.Equal(t, 0, boardCache.ElementsCount())

	messages, err := boardClient.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(initialBoardMessages))

	// 1 cache entry - all messages (in that entry are all 3 messages)
	require.Equal(t, 1, boardCache.ElementsCount())
	messagesFromCacheRaw, found := boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok := messagesFromCacheRaw.([]*Message)
	require.True(t, ok)
	require.Len(t, messagesFromCache, len(initialBoardMessages))

	funcCallsLog := boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 3)
	assert.Equal(t, FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, FuncSet, funcCallsLog[1])
	assert.Equal(t, FuncGetHit, funcCallsLog[2])

	boardCache.ClearFunctionCallsLog()

	// called again - should get it from cache right away
	messages, err = boardClient.AllMessagesCache(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(initialBoardMessages))

	require.Equal(t, 1, boardCache.ElementsCount())
	messagesFromCacheRaw, found = boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, messagesFromCacheRaw)
	messagesFromCache, ok = messagesFromCacheRaw.([]*Message)
	require.True(t, ok)
	assert.Len(t, messagesFromCache, len(initialBoardMessages))

	funcCallsLog = boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 2)
	assert.Equal(t, FuncGetHit, funcCallsLog[0])
	assert.Equal(t, FuncGetHit, funcCallsLog[1])
}

func TestBoard_AllMessages(t *testing.T) {
	boardClient, _, _, initialBoardMessages := getTestBoardClient()

	messages, err := boardClient.AllMessages(false)
	require.NoError(t, err)
	assert.Len(t, messages, len(initialBoardMessages))

	messages, err = boardClient.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, messages, len(initialBoardMessages))
	// sorted by timestamp
	assert.Equal(t, 3, messages[0].ID)
	assert.Equal(t, 2, messages[1].ID)
	assert.Equal(t, 0, messages[2].ID)
	assert.Equal(t, 4, messages[3].ID)
	assert.Equal(t, 1, messages[4].ID)
}

func TestBoard_DeleteMessage(t *testing.T) {
	boardClient, _, _, initialBoardMessages := getTestBoardClient()

	// non existent message
	removed, err := boardClient.DeleteMessage("100")
	require.NoError(t, err)
	assert.False(t, removed)

	// existent message
	messagesCount, err := boardClient.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(initialBoardMessages), messagesCount)

	removed, err = boardClient.DeleteMessage("1")
	require.NoError(t, err)
	assert.True(t, removed)

	messagesCount, err = boardClient.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(initialBoardMessages)-1, messagesCount)
}

func TestBoard_SetAllMessagesCacheFromAero(t *testing.T) {
	boardClient, boardCache, _, initialBoardMessages := getTestBoardClient()

	// cache empty
	require.Equal(t, 0, boardCache.ElementsCount())

	assert.NoError(t, boardClient.SetAllMessagesCacheFromAero())

	// cache filled
	require.Equal(t, 1, boardCache.ElementsCount())
	allMessagesFromCache, found := boardCache.Get(AllMessagesCacheKey)
	require.True(t, found)
	require.NotNil(t, allMessagesFromCache)
	allMessages, ok := allMessagesFromCache.([]*Message)
	require.True(t, ok)
	assert.Len(t, allMessages, len(initialBoardMessages))
}

func TestBoard_CacheBoardMessages(t *testing.T) {
	boardClient, boardCache, _, _ := getTestBoardClient()

	// cache empty
	require.Equal(t, 0, boardCache.ElementsCount())

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

	boardClient.CacheBoardMessages("messages", messages)

	// cache filled
	require.Equal(t, 1, boardCache.ElementsCount())

	allMessagesFromCache, found := boardCache.Get("messages")
	require.True(t, found)
	require.NotNil(t, allMessagesFromCache)
	allMessages, ok := allMessagesFromCache.([]*Message)
	require.True(t, ok)
	assert.Len(t, allMessages, 2)
}

func TestBoard_InvalidateCaches(t *testing.T) {
	boardClient, boardCache, _, _ := getTestBoardClient()

	// cache empty
	require.Equal(t, 0, boardCache.ElementsCount())

	boardClient.CacheBoardMessages("messages", []*Message{
		{
			ID:        0,
			Author:    "a0",
			Timestamp: time.Now().Unix(),
			Message:   "m0",
		},
	})

	// cache filled
	require.Equal(t, 1, boardCache.ElementsCount())

	boardClient.InvalidateCaches()
	// cache empty
	require.Equal(t, 0, boardCache.ElementsCount())
}

func TestBoard_StoreMessage(t *testing.T) {
	boardClient, _, aeroTestClient, initialBoardMessages := getTestBoardClient()

	messagesCount, err := boardClient.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(initialBoardMessages), messagesCount)

	now := time.Now()
	m1 := Message{
		ID:        len(initialBoardMessages),
		Author:    "ana",
		Timestamp: now.Unix(),
		Message:   "lixo",
	}
	newId1, err := boardClient.NewMessage(m1)
	require.NoError(t, err)
	assert.Equal(t, m1.ID, newId1)

	m2 := Message{
		ID:        len(initialBoardMessages) + 1,
		Author:    "serj",
		Timestamp: now.Add(-time.Hour).Unix(),
		Message:   "a message",
	}
	newId2, err := boardClient.NewMessage(m2)
	require.NoError(t, err)
	assert.Equal(t, m2.ID, newId2)

	messagesCount, err = boardClient.MessagesCount()
	require.NoError(t, err)
	require.Equal(t, len(initialBoardMessages)+2, messagesCount)

	allMessages, err := boardClient.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, allMessages, len(initialBoardMessages)+2)

	m1binMap := aeroTestClient.AeroBinMaps[strconv.Itoa(m1.ID)]
	require.NotNil(t, m1binMap)
	assert.Equal(t, m1.Message, m1binMap["message"])
	m2binMap := aeroTestClient.AeroBinMaps[strconv.Itoa(m2.ID)]
	require.NotNil(t, m2binMap)
	assert.Equal(t, m2.Message, m2binMap["message"])
}

func TestBoard_GetMessagesWithRange(t *testing.T) {
	boardClient, boardCache, _, _ := getTestBoardClient()

	// cache empty at the beginning
	require.Equal(t, 0, boardCache.ElementsCount())

	messages, err := boardClient.GetMessagesWithRange(1, 3)
	require.NoError(t, err)

	// cache empty after - GetMessagesWithRange does not cache atm
	require.Equal(t, 0, boardCache.ElementsCount())
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
	boardClient, boardCache, _, _ := getTestBoardClient()

	// cache empty at the beginning
	require.Equal(t, 0, boardCache.ElementsCount())

	messages, err := boardClient.GetMessagesPage(2, 2)
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
	funcCallsLog := boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 2)
	assert.Equal(t, FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, FuncSet, funcCallsLog[1])

	// size greater than total - get all messages
	messages, err = boardClient.GetMessagesPage(2, 12)
	require.NoError(t, err)
	require.Len(t, messages, 5)

	// page greater than total pages - get last page
	messages, err = boardClient.GetMessagesPage(10, 2)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	// first case again
	messages, err = boardClient.GetMessagesPage(2, 2)
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
	funcCallsLog = boardCache.FunctionCallsLog
	require.Len(t, funcCallsLog, 7)
	assert.Equal(t, FuncGetMiss, funcCallsLog[0])
	assert.Equal(t, FuncSet, funcCallsLog[1])
	assert.Equal(t, FuncGetMiss, funcCallsLog[2])
	assert.Equal(t, FuncSet, funcCallsLog[3])
	assert.Equal(t, FuncGetMiss, funcCallsLog[4])
	assert.Equal(t, FuncSet, funcCallsLog[5])
	assert.Equal(t, FuncGetHit, funcCallsLog[6])
}
