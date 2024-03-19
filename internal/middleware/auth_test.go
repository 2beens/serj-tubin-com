package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2beens/serjtubincom/internal/middleware"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAuthMiddlewareHandler_AuthCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLoginChecker := NewMockloginChecker(ctrl)
	authMiddleware := middleware.NewAuthMiddlewareHandler(
		"gymstatsIOSAppSecret",
		"browserRequestsSecret",
		mockLoginChecker,
	)

	testCases := []struct {
		name               string
		path               string
		method             string
		token              string
		userAgent          string
		authTokenHeader    string
		expectedStatusCode int
		mockIsLogged       bool
		mockIsLoggedErr    error
	}{
		{
			name:               "AllowedPathWithoutToken",
			path:               "/blog/all",
			method:             "GET",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "NotAllowedPathWithoutToken",
			path:               "/admin/panel",
			method:             "GET",
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "ValidToken",
			path:               "/secure/resource",
			method:             "GET",
			token:              "valid-token",
			expectedStatusCode: http.StatusOK,
			mockIsLogged:       true,
		},
		{
			name:               "InvalidToken",
			path:               "/secure/resource",
			method:             "GET",
			token:              "invalid-token",
			expectedStatusCode: http.StatusUnauthorized,
			mockIsLogged:       false,
		},
		{
			name:               "BrowserExtensionRequestValidToken",
			path:               "/netlog/new",
			method:             "POST",
			token:              "browserRequestsSecret",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "BrowserExtensionRequestInvalidToken",
			path:               "/netlog/new",
			method:             "POST",
			token:              "invalid-token",
			expectedStatusCode: http.StatusOK, // Response is OK, but it's a decoy.
		},
		{
			name:               "GymStatsAgentValidToken",
			path:               "/gymstats/some-resource",
			method:             "GET",
			userAgent:          "GymStats/1.0",
			authTokenHeader:    "gymstatsIOSAppSecret",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "GymStatsAgentInvalidToken",
			path:               "/gymstats/some-resource",
			method:             "GET",
			userAgent:          "GymStats/1.0",
			authTokenHeader:    "wrong-token",
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "GymStatsWebClientImageLink",
			path:               "/gymstats/image/123",
			method:             "GET",
			userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "GymStatsWebClientExerciseGet",
			path:               "/gymstats/exercise/123",
			method:             "GET",
			userAgent:          "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			expectedStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			assert.NoError(t, err)
			if tc.token != "" {
				req.Header.Add("X-SERJ-TOKEN", tc.token)
			}
			if tc.authTokenHeader != "" {
				req.Header.Add("Authorization", tc.authTokenHeader)
			}
			if tc.userAgent != "" {
				req.Header.Add("User-Agent", tc.userAgent)
			}

			if tc.path == "/secure/resource" {
				mockLoginChecker.EXPECT().
					IsLogged(gomock.Any(), tc.token).
					Return(tc.mockIsLogged, tc.mockIsLoggedErr).AnyTimes()
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			authMiddleware.AuthCheck()(handler).ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatusCode, rr.Code)
		})
	}
}
