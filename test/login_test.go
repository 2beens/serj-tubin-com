package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/2beens/serjtubincom/internal/misc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) TestLogin() {
	t := s.T()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cases := map[string]struct {
		loginReq           misc.LoginRequest
		expectedStatusCode int
		assertFunc         func(resp *http.Response)
	}{
		"good creds": {
			loginReq: misc.LoginRequest{
				Username: testUsername,
				Password: testPassword,
			},
			expectedStatusCode: http.StatusOK,
			assertFunc: func(resp *http.Response) {
				respBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var loginResp misc.LoginResponse
				require.NoError(t, json.Unmarshal(respBytes, &loginResp))
				assert.NotEmpty(t, loginResp.Token)
			},
		},
		"good creds, then logout": {
			loginReq: misc.LoginRequest{
				Username: testUsername,
				Password: testPassword,
			},
			expectedStatusCode: http.StatusOK,
			assertFunc: func(resp *http.Response) {
				respBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var loginResp misc.LoginResponse
				require.NoError(t, json.Unmarshal(respBytes, &loginResp))
				assert.NotEmpty(t, loginResp.Token)

				req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/a/logout", serverEndpoint), nil)
				require.NoError(t, err)
				req.Header.Set("User-Agent", "test-agent")
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-SERJ-TOKEN", loginResp.Token)

				logoutResp, err := s.httpClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, logoutResp.StatusCode)
				resp.Body.Close()
			},
		},
		"bad password": {
			loginReq: misc.LoginRequest{
				Username: testUsername,
				Password: "bad-password",
			},
			expectedStatusCode: http.StatusBadRequest,
			assertFunc: func(resp *http.Response) {
				respBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				respString := strings.TrimSpace(string(respBytes))
				assert.Equal(t, "login failed", respString)
			},
		},
		"bad username": {
			loginReq: misc.LoginRequest{
				Username: "bad-username",
				Password: testPassword,
			},
			expectedStatusCode: http.StatusBadRequest,
			assertFunc: func(resp *http.Response) {
				respBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				respString := strings.TrimSpace(string(respBytes))
				assert.Equal(t, "login failed", respString)
			},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			loginRequest := misc.LoginRequest{
				Username: tc.loginReq.Username,
				Password: tc.loginReq.Password,
			}
			loginReqJson, err := json.Marshal(loginRequest)
			require.NoError(t, err)

			req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/a/login", serverEndpoint), bytes.NewBuffer(loginReqJson))
			require.NoError(t, err)
			req.Header.Set("User-Agent", "test-agent")
			req.Header.Set("Content-Type", "application/json")

			resp, err := s.httpClient.Do(req)
			require.NoError(t, err)
			require.Equal(t, tc.expectedStatusCode, resp.StatusCode)
			defer resp.Body.Close()

			tc.assertFunc(resp)
		})
	}
}
