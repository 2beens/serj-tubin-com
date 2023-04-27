package misc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"

	testingpkg "github.com/2beens/serjtubincom/pkg/testing"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
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

type testRequestRateLimiter struct {
	// key to limit map
	Limits map[string]int
}

func (l *testRequestRateLimiter) Allow(_ context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error) {
	res := &redis_rate.Result{
		Limit: redis_rate.Limit{
			Rate:   0,
			Burst:  0,
			Period: 0,
		},
		Allowed:    0,
		Remaining:  0,
		RetryAfter: 0,
		ResetAfter: 0,
	}

	foundLimit, ok := l.Limits[key]
	if !ok || foundLimit == 0 {
		return res, nil
	}

	res.Allowed = l.Limits[key]
	l.Limits[key]--

	return res, nil
}

func setupNetlogRouterForTests(
	t *testing.T,
	authService *auth.Service,
	adminUsername, adminPassHash string,
	redisClient *redis.Client,
	reqRateLimiter *testRequestRateLimiter,
	metricsManager *metrics.Manager,
	browserReqSecret string,
) *mux.Router {
	t.Helper()

	r := mux.NewRouter()
	authMiddleware := middleware.NewAuthMiddlewareHandler(
		browserReqSecret,
		auth.NewLoginChecker(time.Hour, redisClient),
	)

	// the same setup as in Server.routerSetup() ... these are not so much of a "unit" tests
	r.Use(middleware.PanicRecovery(metricsManager))
	r.Use(middleware.LogRequest())
	r.Use(middleware.RequestMetrics(metricsManager))
	r.Use(middleware.Cors())
	r.Use(authMiddleware.AuthCheck())
	r.Use(middleware.DrainAndCloseRequest())

	handler := NewHandler(
		nil, nil,
		"dummy", authService, &auth.Admin{
			Username:     adminUsername,
			PasswordHash: adminPassHash,
		},
	)
	handler.SetupRoutes(r, reqRateLimiter, metrics.NewTestManager())

	return r
}

func TestNewMiscHandler(t *testing.T) {
	mainRouter := mux.NewRouter()
	handler := NewHandler(nil, nil, "dummy", &auth.Service{}, &auth.Admin{})
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

func TestLogin(t *testing.T) {
	require.NoError(t, os.Setenv("REDIS_PASS", "<remove>"))
	rdb := testingpkg.GetRedisClientAndCtx(t)
	defer func() {
		assert.NoError(t, rdb.Close())
	}()

	authService := auth.NewAuthService(time.Hour, rdb)
	require.NotNil(t, authService)
	testToken := "test_token"
	randStringFunc := func(s int) (string, error) {
		return testToken, nil
	}
	authService.RandStringFunc = randStringFunc

	username := "testuser"
	password := "testpass"
	passwordHash := "$2a$14$6Gmhg85si2etd3K9oB8nYu1cxfbrdmhkg6wI6OXsa88IF4L2r/L9i" // testpass

	reqRateLimiter := &testRequestRateLimiter{
		Limits: map[string]int{},
	}
	r := setupNetlogRouterForTests(
		t,
		authService,
		username,
		passwordHash,
		rdb,
		reqRateLimiter,
		metrics.NewTestManager(),
		"test",
	)

	reqRateLimiter.Limits["login"] = 1

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/a/login", nil)
	req.PostForm = url.Values{}
	req.PostForm.Add("username", username)
	req.PostForm.Add("password", password)
	req.Header.Set("Origin", "test")

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, fmt.Sprintf(`{"token": "%s"}`, testToken), rr.Body.String())

	// next time fails
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooEarly, rr.Code)
	assert.True(t, strings.HasPrefix(rr.Body.String(), "retry after"))
}

// TODO: other tests
