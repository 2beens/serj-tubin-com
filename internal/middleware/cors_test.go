package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCorsMiddleware(t *testing.T) {
	testCases := []struct {
		name           string
		origin         string
		userAgent      string
		path           string
		expectCors     bool
		expectedStatus int
	}{
		{
			name:           "AllowedOrigin",
			origin:         "https://www.serj-tubin.com",
			expectCors:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "NotAllowedOrigin",
			origin:         "https://www.notallowed.com",
			expectCors:     false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "AllowedUserAgent",
			userAgent:      "GymStats/1.0",
			expectCors:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "NotAllowedUserAgent",
			userAgent:      "UnknownAgent/1.0",
			expectCors:     false,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "PathBasedCorsLinks",
			path:           "/link/1234",
			expectCors:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PathBasedCorsGymStatsImages",
			path:           "/gymstats/image/1234",
			expectCors:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PathBasedCorsUnknownPath",
			userAgent:      "unknown-agent",
			path:           "/unknown/path",
			expectCors:     false,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req, err := http.NewRequest("GET", tc.path, nil)
			require.NoError(t, err)
			req.Header.Set("Origin", tc.origin)
			req.Header.Set("User-Agent", tc.userAgent)

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			handler := Cors()(nextHandler)

			handler.ServeHTTP(rr, req)

			if tc.expectCors {
				assert.Equal(t, tc.origin, rr.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Equal(t, tc.expectedStatus, rr.Code, "Unexpected status code")
			}
		})
	}
}
