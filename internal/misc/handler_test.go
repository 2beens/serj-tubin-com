package misc_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/misc"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/go-redis/redis_rate/v9"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// use TestMain(m *testing.M) { ... } for
// global set-up/tear-down for all the tests in a package
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
	)
}

func TestNewMiscHandler(t *testing.T) {
	mainRouter := mux.NewRouter()
	handler := misc.NewHandler(nil, nil, "dummy", &auth.Service{})
	handler.SetupRoutes(mainRouter, nil, metrics.NewTestManager())
	require.NotNil(t, handler)
	require.NotNil(t, mainRouter)

	for caseName, route := range map[string]struct {
		name   string
		path   string
		method string
	}{
		"route-get": {
			name:   "root",
			path:   "/",
			method: "GET",
		},
		"route-post": {
			name:   "root",
			path:   "/",
			method: "POST",
		},
		"route-options": {
			name:   "root",
			path:   "/",
			method: "OPTIONS",
		},
		"quote": {
			name:   "quote",
			path:   "/quote/random",
			method: "GET",
		},
		"whereami": {
			name:   "whereami",
			path:   "/whereami",
			method: "GET",
		},
		"myip": {
			name:   "myip",
			path:   "/myip",
			method: "GET",
		},
		"version": {
			name:   "version",
			path:   "/version",
			method: "GET",
		},
		"login": {
			name:   "login",
			path:   "/a/login",
			method: "POST",
		},
		"logout": {
			name:   "logout",
			path:   "/a/logout",
			method: "GET",
		},
		"logout-otions": {
			name:   "logout",
			path:   "/a/logout",
			method: "OPTIONS",
		},
	} {
		t.Run(caseName, func(t *testing.T) {
			req, err := http.NewRequest(route.method, route.path, nil)
			require.NoError(t, err)

			routeMatch := &mux.RouteMatch{}
			route := mainRouter.Get(route.name)
			require.NotNil(t, route)
			isMatch := route.Match(req, routeMatch)
			assert.True(t, isMatch, caseName)
		})
	}
}

func TestHandler_VersionAndRoot(t *testing.T) {
	ctrl := gomock.NewController(t)
	authServiceMock := NewMockauthService(ctrl)

	handler := misc.NewHandler(
		nil,
		nil,
		"dummy-version-info",
		authServiceMock,
	)

	r := mux.NewRouter()
	handler.SetupRoutes(r, nil, metrics.NewTestManager())

	req, err := http.NewRequest("GET", "/version", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "dummy-version-info", rr.Body.String())

	req, err = http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "I'm OK, thanks ;)", rr.Body.String())
}

func TestHandler_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	authServiceMock := NewMockauthService(ctrl)
	rateLimiterMock := NewMockRequestRateLimiter(ctrl)

	handler := misc.NewHandler(
		nil,
		nil,
		"dummy-version-info",
		authServiceMock,
	)

	r := mux.NewRouter()
	handler.SetupRoutes(
		r,
		rateLimiterMock,
		metrics.NewTestManager(),
	)

	loginRequest := auth.Credentials{
		Username: "test-username",
		Password: "test-password",
	}
	loginRequestJson, err := json.Marshal(loginRequest)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/a/login", bytes.NewReader(loginRequestJson))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "test")

	rateLimiterMock.EXPECT().
		Allow(gomock.Any(), "login", gomock.Any()).
		Return(&redis_rate.Result{
			Allowed: 10,
		}, nil)

	authServiceMock.EXPECT().
		Login(gomock.Any(), loginRequest, gomock.Any()).
		Return("test-token", nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var loginResponse misc.LoginResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &loginResponse))
	assert.Equal(t, "test-token", loginResponse.Token)
}

// TODO: other tests
