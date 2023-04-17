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
	// Do stuff BEFORE the tests
	m.Run()

	// do stuff AFTER the tests
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
	)
}

func TestNewNetlogHandler(t *testing.T) {
	r := mux.NewRouter()
	router := r.PathPrefix("/netlog").Subrouter()
	netlogApi := NewTestApi()
	m := metrics.NewTestManager()
	handler := NewHandler(netlogApi, m, "", nil)
	handler.SetupRoutes(router)
	require.NotNil(t, handler)
	require.NotNil(t, router)

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
			route := router.Get(route.name)
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
	netlogApi := NewTestApi()

	r := mux.NewRouter()
	m := metrics.NewTestManager()
	handler := NewHandler(netlogApi, m, browserReqSecret, loginChecker)
	handler.SetupRoutes(r)
	require.NotNil(t, handler)
	require.NotNil(t, r)

	req, err := http.NewRequest("GET", "/", nil)
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

	browserReqSecret := "beer"
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := NewTestApi()

	r := mux.NewRouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	m := metrics.NewTestManager()
	handler := NewHandler(netlogApi, m, browserReqSecret, loginChecker)
	handler.SetupRoutes(netlogRouter)
	require.NotNil(t, handler)
	require.NotNil(t, r)
	require.NotNil(t, netlogRouter)

	req, err := http.NewRequest("GET", "/netlog/", nil)
	require.NoError(t, err)
	// we remove the auth token:
	//req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	netlogRouter.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var visits []*Visit
	err = json.Unmarshal(rr.Body.Bytes(), &visits)
	require.Error(t, err)
	require.Nil(t, visits)
}

func TestNetlogHandler_handleGetAll(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	browserReqSecret := "beer"
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := NewTestApi()

	now := time.Now()
	visit0 := Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := Visit{
		Id:        1,
		Title:     "test title 1",
		Source:    "chrome",
		URL:       "test:url:1",
		Timestamp: now,
	}
	netlogApi.Visits[0] = visit0
	netlogApi.Visits[1] = visit1

	r := mux.NewRouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	m := metrics.NewTestManager()
	handler := NewHandler(netlogApi, m, browserReqSecret, loginChecker)
	handler.SetupRoutes(netlogRouter)
	require.NotNil(t, handler)
	require.NotNil(t, r)
	require.NotNil(t, netlogRouter)

	req, err := http.NewRequest("GET", "/netlog/", nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	netlogRouter.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var visits []*Visit
	err = json.Unmarshal(rr.Body.Bytes(), &visits)
	require.NoError(t, err)
	require.NotNil(t, visits)
	assert.Len(t, visits, 2)
}

func TestNetlogHandler_handleNewVisit_invalidToken(t *testing.T) {
	db, _ := redismock.NewClientMock()

	browserReqSecret := "rakija"
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := NewTestApi()

	now := time.Now()
	visit0 := Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := Visit{
		Id:        1,
		Title:     "test title 1",
		Source:    "chrome",
		URL:       "test:url:1",
		Timestamp: now,
	}
	netlogApi.Visits[0] = visit0
	netlogApi.Visits[1] = visit1

	assert.Len(t, netlogApi.Visits, 2)

	r := mux.NewRouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	m := metrics.NewTestManager()
	handler := NewHandler(netlogApi, m, browserReqSecret, loginChecker)
	handler.SetupRoutes(netlogRouter)
	require.NotNil(t, handler)
	require.NotNil(t, r)
	require.NotNil(t, netlogRouter)

	req, err := http.NewRequest("POST", "/netlog/new", nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "beer")
	rr := httptest.NewRecorder()

	jsTimestamp := 1612622746987
	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense Title")
	req.PostForm.Add("source", "safari")
	req.PostForm.Add("url", "https://hypofriend.de/en/mortgage-tips/first-time-buyers")
	req.PostForm.Add("timestamp", strconv.Itoa(jsTimestamp))

	netlogRouter.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))

	resp := rr.Body.Bytes()
	assert.Equal(t, "added", string(resp)) // this is a false positive "added"

	// visits len is unchanged
	assert.Len(t, netlogApi.Visits, 2)

	assert.Equal(t, float64(0), testutil.ToFloat64(m.CounterNetlogVisits))
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

				return req
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			db, _ := redismock.NewClientMock()

			browserReqSecret := "beer"
			loginChecker := auth.NewLoginChecker(time.Hour, db)
			netlogApi := NewTestApi()

			now := time.Now()
			visit0 := Visit{
				Id:        0,
				Title:     "test title 0",
				Source:    "chrome",
				URL:       "test:url:0",
				Timestamp: now,
			}
			visit1 := Visit{
				Id:        1,
				Title:     "test title 1",
				Source:    "chrome",
				URL:       "test:url:1",
				Timestamp: now,
			}
			netlogApi.Visits[0] = visit0
			netlogApi.Visits[1] = visit1

			assert.Len(t, netlogApi.Visits, 2)

			r := mux.NewRouter()
			netlogRouter := r.PathPrefix("/netlog").Subrouter()
			m := metrics.NewTestManager()
			handler := NewHandler(netlogApi, m, browserReqSecret, loginChecker)
			handler.SetupRoutes(netlogRouter)
			require.NotNil(t, handler)
			require.NotNil(t, r)
			require.NotNil(t, netlogRouter)

			newVisitReq := newVisitRequest{
				Title:     "Nonsense Title",
				Source:    "safari",
				Device:    "super-cool-pc",
				URL:       "https://hypofriend.de/en/mortgage-tips/first-time-buyers",
				Timestamp: 1612622746987,
			}
			req := tc.req(t, newVisitReq)
			rr := httptest.NewRecorder()

			netlogRouter.ServeHTTP(rr, req)
			require.Equal(t, http.StatusCreated, rr.Code)
			assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))

			resp := rr.Body.Bytes()
			assert.Equal(t, "added", string(resp))

			assert.Len(t, netlogApi.Visits, 3)
			addedVisit, ok := netlogApi.Visits[2]
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

func TestNetlogHandler_handleGetPage(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := NewTestApi()

	now := time.Now()
	visit0 := Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := Visit{
		Id:        1,
		Title:     "test title 1",
		Source:    "chrome",
		URL:       "test:url:1",
		Timestamp: now,
	}

	netlogApi.Visits[0] = visit0
	netlogApi.Visits[1] = visit1

	for id := 2; id <= 8; id++ {
		netlogApi.Visits[id] = Visit{
			Id:        id,
			Title:     fmt.Sprintf("test title %d", id),
			Source:    "safari",
			URL:       fmt.Sprintf("test:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	for id := 9; id <= 12; id++ {
		netlogApi.Visits[id] = Visit{
			Id:        id,
			Title:     fmt.Sprintf("other title %d", id),
			Source:    "safari",
			URL:       fmt.Sprintf("other:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	for id := 12; id <= 15; id++ {
		netlogApi.Visits[id] = Visit{
			Id:        id,
			Title:     fmt.Sprintf("test title %d", id),
			Source:    "pc",
			URL:       fmt.Sprintf("test:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	assert.Len(t, netlogApi.Visits, 16)

	r := mux.NewRouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	m := metrics.NewTestManager()
	handler := NewHandler(netlogApi, m, "browserReqSecret", loginChecker)
	handler.SetupRoutes(netlogRouter)
	require.NotNil(t, handler)
	require.NotNil(t, r)
	require.NotNil(t, netlogRouter)

	req, err := http.NewRequest("GET", "/netlog/s/safari/f/url/search/test:url/page/2/size/3", nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	netlogRouter.ServeHTTP(rr, req)
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
