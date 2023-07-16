package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2beens/serjtubincom/internal/misc"
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

	t.Run("rate limiting", func(t *testing.T) {
		// simulate login requests brute force attack
		loginRequest := misc.LoginRequest{
			Username: "test-user",
			Password: "test-pass",
		}
		loginReqJson, err := json.Marshal(loginRequest)
		require.NoError(t, err)

		// config is set to allow 10 login attempts per minute, so after 10th attempt we should get 429
		// but first, do a redis cleanup
		require.NoError(t, s.redisDataCleanup(ctx))

		for i := 1; i <= 15; i++ {
			req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/a/login", serverEndpoint), bytes.NewBuffer(loginReqJson))
			require.NoError(t, err)
			req.Header.Set("User-Agent", "test-agent")
			req.Header.Set("Content-Type", "application/json")

			resp, err := s.httpClient.Do(req)
			require.NoError(t, err)

			if i <= 10 {
				require.Equal(t, http.StatusBadRequest, resp.StatusCode, "iteration: %d", i)
				assert.Empty(t, resp.Header.Get("Retry-After"), "iteration: %d", i)
			} else {
				require.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "iteration: %d", i)
				retryAfter, err := strconv.ParseFloat(resp.Header.Get("Retry-After"), 64)
				require.NoError(t, err, "iteration: %d", i)
				assert.True(t, retryAfter > 0, "iteration: %d", i)
			}

			assert.NoError(t, resp.Body.Close())
		}

		require.NoError(t, s.redisDataCleanup(ctx))
	})
}
