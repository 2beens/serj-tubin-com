package aerospike

import (
	"errors"
	"fmt"
	"testing"
	"time"

	as "github.com/aerospike/aerospike-client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// use TestMain(m *testing.M) { ... } for
// global set-up/tear-down for all the tests in a package
func TestMain(m *testing.M) {
	// Do stuff BEFORE the tests
	m.Run()

	// do stuff AFTER the tests
	goleak.VerifyTestMain(m)
}

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

func TestNewBoardAeroClient_NewAeroClientFailed(t *testing.T) {
	dummyErr := errors.New("dummy err")
	newAerospikeClientFunc := func(hostname string, port int) (*as.Client, error) {
		return nil, dummyErr
	}

	boardClient, err := newDefaultBoardAeroClient("testhost", 9000, "testnamespace", "testset", newAerospikeClientFunc)
	assert.NoError(t, err)
	require.NotNil(t, boardClient)

	checkConnErr := boardClient.CheckConnection()
	assert.True(t, errors.Is(checkConnErr, dummyErr), fmt.Sprintf("expecting err: [%s], but got: [%s]", dummyErr, checkConnErr))
}

func TestBoardAeroClient_CheckConnection(t *testing.T) {
	sem := make(chan struct{})

	newAerospikeClientFunc := func(hostname string, port int) (*as.Client, error) {
		<-sem
		return &as.Client{}, nil
	}

	boardClient, err := newDefaultBoardAeroClient("testhost", 9000, "testnamespace", "testset", newAerospikeClientFunc)
	assert.NoError(t, err)
	require.NotNil(t, boardClient)

	assert.Never(t, func() bool {
		return boardClient.WaitForReady(200*time.Millisecond) == nil
	}, time.Second, 100*time.Millisecond)

	err = boardClient.CheckConnection()
	require.Error(t, err)
	assert.Equal(t, "aero client already connecting", err.Error())

	sem <- struct{}{}
	assert.Eventually(t, func() bool {
		return boardClient.WaitForReady(200*time.Millisecond) == nil
	}, time.Second, 100*time.Millisecond)

	// TODO: first abstract the aero client away from the board api, then test this
	//assert.NoError(t, boardClient.CheckConnection())
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
