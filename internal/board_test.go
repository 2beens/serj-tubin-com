package internal

import (
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoard_CheckAero(t *testing.T) {
	board, err := NewBoard(nil, "aero-test")
	require.Equal(t, aerospike.ErrAeroClientNil, err)
	assert.Nil(t, board)
}

func TestBoard_AllMessages(t *testing.T) {
	aeroTestClient := aerospike.NewBoardAeroTestClient()

	now := time.Now()

	err := aeroTestClient.Put("0", aerospike.AeroBinMap{
		"id":        0,
		"author":    "serj",
		"message":   "test message blabla",
		"timestamp": now.Add(-time.Hour).Unix(),
	})
	require.NoError(t, err)
	err = aeroTestClient.Put("1", aerospike.AeroBinMap{
		"id":        1,
		"author":    "serj",
		"message":   "test message gragra",
		"timestamp": now.Unix(),
	})
	err = aeroTestClient.Put("2", aerospike.AeroBinMap{
		"id":        2,
		"author":    "ana",
		"message":   "test message aaaaa",
		"timestamp": now.Add(-2 * time.Hour).Unix(),
	})
	require.NoError(t, err)

	board, err := NewBoard(aeroTestClient, "aero-test")
	require.NoError(t, err)
	assert.NotNil(t, board)

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
