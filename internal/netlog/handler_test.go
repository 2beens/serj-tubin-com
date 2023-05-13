package netlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/go-redis/redismock/v8"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// use TestMain(m *testing.M) { ... } for
// global set-up/tear-down for all the tests in a package (enough to place it in one of the test
// files of the package)
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
	)
}

func setupNetlogRouterForTests(
	t *testing.T,
	repo netlogRepo,
	metricsManager *metrics.Manager,
	loginChecker *auth.LoginChecker,
	browserReqSecret string,
) *mux.Router {
	t.Helper()

	r := mux.NewRouter()
	authMiddleware := middleware.NewAuthMiddlewareHandler(
		"n/a",
		browserReqSecret,
		loginChecker,
	)

	// the same setup as in Server.routerSetup() ... these are not so much of a "unit" tests
	r.Use(middleware.PanicRecovery(metricsManager))
	r.Use(middleware.LogRequest())
	r.Use(middleware.RequestMetrics(metricsManager))
	r.Use(middleware.Cors())
	r.Use(authMiddleware.AuthCheck())
	r.Use(middleware.DrainAndCloseRequest())

	handler := NewHandler(repo, metricsManager, browserReqSecret, loginChecker)
	handler.SetupRoutes(r)

	return r
}

func TestNewNetlogHandler(t *testing.T) {
	r := mux.NewRouter()
	repo := NewRepoMock()
	m := metrics.NewTestManager()
	handler := NewHandler(repo, m, "", nil)
	handler.SetupRoutes(r)

	for caseName, route := range map[string]struct {
		name   string
		path   string
		method string
	}{
		"new-visit-post": {
			name:   "new-visit",
			path:   "/netlog/new",
			method: "POST",
		},
		"new-visit-options": {
			name:   "new-visit",
			path:   "/netlog/new",
			method: "OPTIONS",
		},
		"get-last-get": {
			name:   "get-last",
			path:   "/netlog/",
			method: "GET",
		},
		"get-last-options": {
			name:   "get-last",
			path:   "/netlog/",
			method: "OPTIONS",
		},
		"get-with-limit": {
			name:   "get-with-limit",
			path:   "/netlog/limit/{limit}",
			method: "GET",
		},
		"visits-page": {
			name:   "visits-page",
			path:   "/netlog/s/{source}/f/{field}/page/{page}/size/{size}",
			method: "GET",
		},
		"search-page": {
			name:   "search-page",
			path:   "/netlog/s/{source}/f/{field}/search/{keywords}/page/{page}/size/{size}",
			method: "GET",
		},
	} {
		t.Run(caseName, func(t *testing.T) {
			req, err := http.NewRequest(route.method, route.path, nil)
			require.NoError(t, err)

			routeMatch := &mux.RouteMatch{}
			route := r.Get(route.name)
			require.NotNil(t, route)
			isMatch := route.Match(req, routeMatch)
			assert.True(t, isMatch, caseName)
		})
	}
}

func TestNetlogHandler_handleGetAll_Empty(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	browserReqSecret := "beer"
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	repo := NewRepoMock()
	repo.Visits = make(map[int]Visit, 0)

	r := mux.NewRouter()
	m := metrics.NewTestManager()
	handler := NewHandler(repo, m, browserReqSecret, loginChecker)
	handler.SetupRoutes(r)
	require.NotNil(t, handler)
	require.NotNil(t, r)

	req, err := http.NewRequest("GET", "/netlog/", nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var visits []*Visit
	err = json.Unmarshal(rr.Body.Bytes(), &visits)
	require.NoError(t, err)
	require.NotNil(t, visits)
	assert.Empty(t, visits)
}

func TestNetlogHandler_handleGetAll_Unauthorized(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	repo := NewRepoMock()
	m := metrics.NewTestManager()
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	browserReqSecret := "beer"
	r := setupNetlogRouterForTests(t, repo, m, loginChecker, browserReqSecret)

	req, err := http.NewRequest("GET", "/netlog/", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	// we remove the auth token, i.e. it's not set:
	//req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "no can do\n", rr.Body.String())
}

func TestNetlogHandler_handleGetAll(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	repo := NewRepoMock()
	m := metrics.NewTestManager()
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	browserReqSecret := "beer"
	r := setupNetlogRouterForTests(t, repo, m, loginChecker, browserReqSecret)

	req, err := http.NewRequest("GET", "/netlog/", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var visits []*Visit
	err = json.Unmarshal(rr.Body.Bytes(), &visits)
	require.NoError(t, err)
	require.NotNil(t, visits)
	assert.Len(t, visits, 2)
}

func TestNetlogHandler_handleNewVisit_invalidToken(t *testing.T) {
	testCases := map[string]struct {
		req func(t *testing.T, newVisitReq newVisitRequest) *http.Request
	}{
		"post form payload": {
			req: func(t *testing.T, newVisitReq newVisitRequest) *http.Request {
				req, err := http.NewRequest("POST", "/netlog/new", nil)
				require.NoError(t, err)
				req.Header.Set("Origin", "test")
				req.Header.Set("X-SERJ-TOKEN", "beer")

				req.PostForm = url.Values{}
				req.PostForm.Add("title", "Nonsense Title")
				req.PostForm.Add("source", "safari")
				req.PostForm.Add("url", "https://hypofriend.de/en/mortgage-tips/first-time-buyers")
				req.PostForm.Add("timestamp", strconv.Itoa(1612622746987))

				return req
			},
		},
		"json payload": {
			req: func(t *testing.T, newVisitReq newVisitRequest) *http.Request {
				reqBytes, err := json.Marshal(newVisitRequest{
					Title:     newVisitReq.Title,
					Source:    newVisitReq.Source,
					Device:    newVisitReq.Device,
					URL:       newVisitReq.URL,
					Timestamp: newVisitReq.Timestamp,
				})
				require.NoError(t, err)

				req, err := http.NewRequest("POST", "/netlog/new", bytes.NewBuffer(reqBytes))
				require.NoError(t, err)
				req.Header.Set("X-SERJ-TOKEN", "beer")
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", "test")

				return req
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			db, _ := redismock.NewClientMock()

			repo := NewRepoMock()
			m := metrics.NewTestManager()
			loginChecker := auth.NewLoginChecker(time.Hour, db)
			browserReqSecret := "rakija"
			r := setupNetlogRouterForTests(t, repo, m, loginChecker, browserReqSecret)

			newVisitReq := newVisitRequest{
				Title:     "Nonsense Title",
				Source:    "safari",
				Device:    "super-cool-pc",
				URL:       "https://hypofriend.de/en/mortgage-tips/first-time-buyers",
				Timestamp: 1612622746987,
			}
			req := tc.req(t, newVisitReq)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))

			resp := rr.Body.Bytes()
			assert.Equal(t, "added", string(resp)) // this is a false positive "added"

			// visits len is unchanged
			assert.Len(t, repo.Visits, 2)
			assert.Equal(t, float64(0), testutil.ToFloat64(m.CounterNetlogVisits))
		})
	}
}

func TestNetlogHandler_handleNewVisit_validToken(t *testing.T) {
	testCases := map[string]struct {
		req func(t *testing.T, newVisitReq newVisitRequest) *http.Request
	}{
		"post form payload": {
			req: func(t *testing.T, newVisitReq newVisitRequest) *http.Request {
				req, err := http.NewRequest("POST", "/netlog/new", nil)
				require.NoError(t, err)
				req.Header.Set("X-SERJ-TOKEN", "beer")
				req.Header.Set("Origin", "test")

				req.PostForm = url.Values{}
				req.PostForm.Add("title", newVisitReq.Title)
				req.PostForm.Add("source", newVisitReq.Source)
				req.PostForm.Add("device", newVisitReq.Device)
				req.PostForm.Add("url", newVisitReq.URL)
				req.PostForm.Add("timestamp", fmt.Sprintf("%d", newVisitReq.Timestamp))

				return req
			},
		},
		"json payload": {
			req: func(t *testing.T, newVisitReq newVisitRequest) *http.Request {
				reqBytes, err := json.Marshal(newVisitRequest{
					Title:     newVisitReq.Title,
					Source:    newVisitReq.Source,
					Device:    newVisitReq.Device,
					URL:       newVisitReq.URL,
					Timestamp: newVisitReq.Timestamp,
				})
				require.NoError(t, err)

				req, err := http.NewRequest("POST", "/netlog/new", bytes.NewBuffer(reqBytes))
				require.NoError(t, err)
				req.Header.Set("X-SERJ-TOKEN", "beer")
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", "test")

				return req
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// TODO: add integration tests instead of this approach
			db, _ := redismock.NewClientMock()
			repo := NewRepoMock()
			m := metrics.NewTestManager()
			loginChecker := auth.NewLoginChecker(time.Hour, db)
			browserReqSecret := "beer"
			r := setupNetlogRouterForTests(t, repo, m, loginChecker, browserReqSecret)

			newVisitReq := newVisitRequest{
				Title:     "Nonsense Title",
				Source:    "safari",
				Device:    "super-cool-pc",
				URL:       "https://hypofriend.de/en/mortgage-tips/first-time-buyers",
				Timestamp: 1612622746987,
			}
			req := tc.req(t, newVisitReq)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)
			require.Equal(t, http.StatusCreated, rr.Code)
			assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))

			resp := rr.Body.Bytes()
			assert.Equal(t, "added", string(resp))

			assert.Len(t, repo.Visits, 3)
			addedVisit, ok := repo.Visits[2]
			require.True(t, ok)
			require.NotNil(t, addedVisit)
			assert.Equal(t, newVisitReq.Title, addedVisit.Title)
			assert.Equal(t, newVisitReq.Source, addedVisit.Source)
			assert.Equal(t, newVisitReq.URL, addedVisit.URL)
			assert.Equal(t, newVisitReq.Device, addedVisit.Device)
			assert.Equal(t, time.Unix(int64(newVisitReq.Timestamp)/1000, 0), addedVisit.Timestamp)

			assert.Equal(t, float64(1), testutil.ToFloat64(m.CounterNetlogVisits))
		})
	}
}

func TestNetlogHandler_handleNewVisit_options_validToken(t *testing.T) {
	db, _ := redismock.NewClientMock()
	repo := NewRepoMock()
	m := metrics.NewTestManager()
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	browserReqSecret := "beer"
	r := setupNetlogRouterForTests(t, repo, m, loginChecker, browserReqSecret)

	req, err := http.NewRequest("OPTIONS", "/netlog/new", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	assert.Equal(t, "", rr.Body.String())
	assert.Len(t, repo.Visits, 2)
	assert.Equal(t, float64(0), testutil.ToFloat64(m.CounterNetlogVisits))
}

func TestNetlogHandler_handleGetPage(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	repo := NewRepoMock()
	m := metrics.NewTestManager()
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	browserReqSecret := "beer"
	r := setupNetlogRouterForTests(t, repo, m, loginChecker, browserReqSecret)

	now := time.Now()
	for id := 2; id <= 8; id++ {
		repo.Visits[id] = Visit{
			Id:        id,
			Title:     fmt.Sprintf("test title %d", id),
			Source:    "safari",
			URL:       fmt.Sprintf("test:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	for id := 9; id <= 12; id++ {
		repo.Visits[id] = Visit{
			Id:        id,
			Title:     fmt.Sprintf("other title %d", id),
			Source:    "safari",
			URL:       fmt.Sprintf("other:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	for id := 12; id <= 15; id++ {
		repo.Visits[id] = Visit{
			Id:        id,
			Title:     fmt.Sprintf("test title %d", id),
			Source:    "pc",
			URL:       fmt.Sprintf("test:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	assert.Len(t, repo.Visits, 16)

	req, err := http.NewRequest("GET", "/netlog/s/safari/f/url/search/test:url/page/2/size/3", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	type getVisitPageResp struct {
		Visits []Visit `json:"visits"`
		Total  int     `json:"total"`
	}

	var resp *getVisitPageResp
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotNil(t, resp)
	require.NotNil(t, resp.Visits)
	assert.Equal(t, 7, resp.Total)
	assert.Len(t, resp.Visits, 3)

	assert.Equal(t, 5, resp.Visits[0].Id)
	assert.Equal(t, 6, resp.Visits[1].Id)
	assert.Equal(t, 7, resp.Visits[2].Id)
	assert.Equal(t, "safari", resp.Visits[0].Source)
	assert.Equal(t, "safari", resp.Visits[1].Source)
	assert.Equal(t, "safari", resp.Visits[2].Source)
	assert.Equal(t, "test:url:5", resp.Visits[0].URL)
	assert.Equal(t, "test:url:6", resp.Visits[1].URL)
	assert.Equal(t, "test:url:7", resp.Visits[2].URL)

	assert.Equal(t, float64(0), testutil.ToFloat64(m.CounterNetlogVisits))
}
