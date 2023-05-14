package test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GymStats_HappyPaths(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	suite := newSuite(ctx)
	defer suite.cleanup()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/gymstats/list", serverEndpoint), nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(respBytes))
}
