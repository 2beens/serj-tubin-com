package aerospike

import (
	"testing"
	"time"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBoardAeroClient(t *testing.T) {
	newAerospikeClientFunc := func(hostname string, port int) (*as.Client, error) {
		return &as.Client{}, nil
	}

	boardClient, err := newDefaultBoardAeroClient("testhost", 9000, "testnamespace", "", newAerospikeClientFunc)
	assert.Nil(t, boardClient)
	assert.Equal(t, ErrEmptySet, err)
	boardClient, err = newDefaultBoardAeroClient("testhost", 9000, "", "testset", newAerospikeClientFunc)
	assert.Nil(t, boardClient)
	assert.Equal(t, ErrEmptyNamespace, err)

	boardClient, err = newDefaultBoardAeroClient("testhost", 9000, "testnamespace", "testset", newAerospikeClientFunc)
	assert.NoError(t, err)
	require.NotNil(t, boardClient)

	assert.Eventually(t, func() bool {
		return boardClient.WaitForReady(200*time.Millisecond) == nil
	}, time.Second, 100*time.Millisecond)
}

func TestBoardAeroClient_WaitForReady(t *testing.T) {
	newAerospikeClientFunc := func(hostname string, port int) (*as.Client, error) {
		return &as.Client{}, nil
	}

	boardClient, err := newDefaultBoardAeroClient("testhost", 9000, "testnamespace", "testset", newAerospikeClientFunc)
	assert.NoError(t, err)
	require.NotNil(t, boardClient)

	assert.Eventually(t, func() bool {
		return boardClient.WaitForReady(200*time.Millisecond) == nil
	}, time.Second, 100*time.Millisecond)

	// reset the ready channel
	boardClient.ready = make(chan struct{})

	assert.Never(t, func() bool {
		return boardClient.WaitForReady(200*time.Millisecond) == nil
	}, time.Second, 100*time.Millisecond)
	assert.Eventually(t, func() bool {
		return boardClient.WaitForReady(200*time.Millisecond) == ErrAeroClientConnTimeout
	}, time.Second, 100*time.Millisecond)

	close(boardClient.ready)

	assert.Eventually(t, func() bool {
		return boardClient.WaitForReady(200*time.Millisecond) == nil
	}, time.Second, 100*time.Millisecond)
}
