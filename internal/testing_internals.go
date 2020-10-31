package internal

import (
	"log"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/cache"
)

type testingInternals struct {
	aeroTestClient *aerospike.BoardAeroTestClient
	board          *Board
	boardCache     *cache.BoardTestCache
}

var initialTestMessagesCount int

func newTestingInternals() (*testingInternals, *Board) {
	aeroClient := aerospike.NewBoardAeroTestClient()
	boardCache := cache.NewBoardTestCache()

	board, err := NewBoard(aeroClient, boardCache)
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	if err := board.StoreMessage(BoardMessage{
		ID:        0,
		Author:    "serj",
		Timestamp: now.Add(-time.Hour).Unix(),
		Message:   "test message blabla",
	}); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(BoardMessage{
		ID:        1,
		Author:    "serj",
		Timestamp: now.Unix(),
		Message:   "test message gragra",
	}); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(BoardMessage{
		ID:        2,
		Author:    "ana",
		Timestamp: now.Add(-2 * time.Hour).Unix(),
		Message:   "test message aaaaa",
	}); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(BoardMessage{
		ID:        3,
		Author:    "drago",
		Timestamp: now.Add(-5 * 24 * time.Hour).Unix(),
		Message:   "drago's test message aaaaa sve",
	}); err != nil {
		panic(err)
	}
	if err := board.StoreMessage(BoardMessage{
		ID:        4,
		Author:    "rodjak nenad",
		Timestamp: now.Add(-2 * time.Minute).Unix(),
		Message:   "ja se mislim sta'e bilo",
	}); err != nil {
		panic(err)
	}

	initialTestMessagesCount = board.messagesCount
	boardCache.ClearFunctionCallsLog()

	return &testingInternals{
		aeroTestClient: aeroClient,
		board:          board,
		boardCache:     boardCache,
	}, board
}
