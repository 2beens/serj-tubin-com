package test

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) TestGymStats() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/gymstats/list", serverEndpoint), nil)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", testGymStatsIOSAppSecret)

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "[]", string(respBytes))
}
