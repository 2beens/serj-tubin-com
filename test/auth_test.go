package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/2beens/serjtubincom/internal/misc"

	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) doLogin(ctx context.Context) string {
	t := s.T()
	loginRequest := misc.LoginRequest{
		Username: testUsername,
		Password: testPassword,
	}
	loginReqJson, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/a/login", serverEndpoint), bytes.NewBuffer(loginReqJson))
	require.NoError(t, err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NotEmpty(t, respBytes)

	var loginResp misc.LoginResponse
	require.NoError(t, json.Unmarshal(respBytes, &loginResp))

	return loginResp.Token
}
