package internal

import (
	"testing"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoard_AllMessages(t *testing.T) {
	aeroTestClient := aerospike.NewBoardAeroTestClient()
	board, err := NewBoard(aeroTestClient, "aero-test")
	require.NoError(t, err)
	assert.NotNil(t, board)
}
