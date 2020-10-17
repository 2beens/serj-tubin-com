package internal

import (
	"log"
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

func newTestingInternals() (*TestingInternals, *Board) {
	aeroClient := aerospike.NewBoardAeroTestClient()
	boardCache := cache.NewBoardTestCache()
	board, err := NewBoard(aeroClient, boardCache, "aero-test")
	if err != nil {
		log.Fatal(err)
	}
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

func TestBoard_AllMessages(t *testing.T) {
	internals, board := newTestingInternals()
	now := time.Now()

	err := internals.aeroTestClient.Put("0", aerospike.AeroBinMap{
		"id":        0,
		"author":    "serj",
		"message":   "test message blabla",
		"timestamp": now.Add(-time.Hour).Unix(),
	})
	require.NoError(t, err)
	err = internals.aeroTestClient.Put("1", aerospike.AeroBinMap{
		"id":        1,
		"author":    "serj",
		"message":   "test message gragra",
		"timestamp": now.Unix(),
	})
	err = internals.aeroTestClient.Put("2", aerospike.AeroBinMap{
		"id":        2,
		"author":    "ana",
		"message":   "test message aaaaa",
		"timestamp": now.Add(-2 * time.Hour).Unix(),
	})
	require.NoError(t, err)

	messages, err := board.AllMessages(true)
	require.NoError(t, err)
	assert.Len(t, messages, 3)
	// sorted by timestamp
	assert.Equal(t, 2, messages[0].ID)
	assert.Equal(t, 0, messages[1].ID)
	assert.Equal(t, 1, messages[2].ID)

	messages, err = board.AllMessages(false)
	require.NoError(t, err)
	assert.Len(t, messages, 3)
	// not sorted by timestamp
	assert.Equal(t, 0, messages[0].ID)
	assert.Equal(t, 1, messages[1].ID)
	assert.Equal(t, 2, messages[2].ID)
}
