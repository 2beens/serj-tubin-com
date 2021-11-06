package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/go-redis/redismock/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetlogHandler(t *testing.T) {
	r := mux.NewRouter()
	router := r.PathPrefix("/netlog").Subrouter()
	netlogApi := netlog.NewTestApi()
	instr := instrumentation.NewTestInstrumentation()
	handler := NewNetlogHandler(router, netlogApi, instr, "", nil)
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
	netlogApi := netlog.NewTestApi()

	r := mux.NewRouter()
	instr := instrumentation.NewTestInstrumentation()
	handler := NewNetlogHandler(r, netlogApi, instr, browserReqSecret, loginChecker)
	require.NotNil(t, handler)
	require.NotNil(t, r)

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var visits []*netlog.Visit
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
	netlogApi := netlog.NewTestApi()

	r := mux.NewRouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	instr := instrumentation.NewTestInstrumentation()
	handler := NewNetlogHandler(netlogRouter, netlogApi, instr, browserReqSecret, loginChecker)
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

	var visits []*netlog.Visit
	err = json.Unmarshal(rr.Body.Bytes(), &visits)
	require.Error(t, err)
	require.Nil(t, visits)
}

func TestNetlogHandler_handleGetAll(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	browserReqSecret := "beer"
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := netlog.NewTestApi()

	now := time.Now()
	visit0 := netlog.Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := netlog.Visit{
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
	instr := instrumentation.NewTestInstrumentation()
	handler := NewNetlogHandler(netlogRouter, netlogApi, instr, browserReqSecret, loginChecker)
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

	var visits []*netlog.Visit
	err = json.Unmarshal(rr.Body.Bytes(), &visits)
	require.NoError(t, err)
	require.NotNil(t, visits)
	assert.Len(t, visits, 2)
}

func TestNetlogHandler_handleNewVisit_invalidToken(t *testing.T) {
	db, _ := redismock.NewClientMock()

	browserReqSecret := "rakija"
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := netlog.NewTestApi()

	now := time.Now()
	visit0 := netlog.Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := netlog.Visit{
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
	instr := instrumentation.NewTestInstrumentation()
	handler := NewNetlogHandler(netlogRouter, netlogApi, instr, browserReqSecret, loginChecker)
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
	assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

	resp := rr.Body.Bytes()
	assert.Equal(t, "added", string(resp)) // this is a false positive "added"

	// visits len is unchanged
	assert.Len(t, netlogApi.Visits, 2)

	assert.Equal(t, float64(0), testutil.ToFloat64(instr.CounterNetlogVisits))
}

func TestNetlogHandler_handleNewVisit_validToken(t *testing.T) {
	db, _ := redismock.NewClientMock()

	browserReqSecret := "beer"
	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := netlog.NewTestApi()

	now := time.Now()
	visit0 := netlog.Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := netlog.Visit{
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
	instr := instrumentation.NewTestInstrumentation()
	handler := NewNetlogHandler(netlogRouter, netlogApi, instr, browserReqSecret, loginChecker)
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
	assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))

	resp := rr.Body.Bytes()
	assert.Equal(t, "added", string(resp))

	assert.Len(t, netlogApi.Visits, 3)
	addedVisit, ok := netlogApi.Visits[2]
	require.True(t, ok)
	require.NotNil(t, addedVisit)
	assert.Equal(t, req.PostForm.Get("title"), addedVisit.Title)
	assert.Equal(t, req.PostForm.Get("source"), addedVisit.Source)
	assert.Equal(t, req.PostForm.Get("url"), addedVisit.URL)
	assert.Equal(t, time.Unix(int64(jsTimestamp)/1000, 0), addedVisit.Timestamp)

	assert.Equal(t, float64(1), testutil.ToFloat64(instr.CounterNetlogVisits))
}

func TestNetlogHandler_handleGetPage(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	loginChecker := auth.NewLoginChecker(time.Hour, db)
	netlogApi := netlog.NewTestApi()

	now := time.Now()
	visit0 := netlog.Visit{
		Id:        0,
		Title:     "test title 0",
		Source:    "chrome",
		URL:       "test:url:0",
		Timestamp: now,
	}
	visit1 := netlog.Visit{
		Id:        1,
		Title:     "test title 1",
		Source:    "chrome",
		URL:       "test:url:1",
		Timestamp: now,
	}

	netlogApi.Visits[0] = visit0
	netlogApi.Visits[1] = visit1

	for id := 2; id <= 8; id++ {
		netlogApi.Visits[id] = netlog.Visit{
			Id:        id,
			Title:     fmt.Sprintf("test title %d", id),
			Source:    "safari",
			URL:       fmt.Sprintf("test:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	for id := 9; id <= 12; id++ {
		netlogApi.Visits[id] = netlog.Visit{
			Id:        id,
			Title:     fmt.Sprintf("other title %d", id),
			Source:    "safari",
			URL:       fmt.Sprintf("other:url:%d", id),
			Timestamp: now.Add(time.Duration(id) * time.Hour),
		}
	}

	for id := 12; id <= 15; id++ {
		netlogApi.Visits[id] = netlog.Visit{
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
	instr := instrumentation.NewTestInstrumentation()
	handler := NewNetlogHandler(netlogRouter, netlogApi, instr, "browserReqSecret", loginChecker)
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
		Visits []netlog.Visit `json:"visits"`
		Total  int            `json:"total"`
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

	assert.Equal(t, float64(0), testutil.ToFloat64(instr.CounterNetlogVisits))
}
