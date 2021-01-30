package internal

import (
	"net/http"
	"testing"

	"encoding/json"
	"net/http/httptest"

	"time"

	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNetlogHandler(t *testing.T) {
	router := mux.NewRouter()
	netlogApi := netlog.NewTestApi()
	handler := NewNetlogHandler(router, netlogApi, "", nil)
	require.NotNil(t, handler)
	require.NotNil(t, router)

	for caseName, route := range map[string]struct {
		name   string
		path   string
		method string
	}{
		"new-visit-post": {
			name:   "new-visit",
			path:   "/new",
			method: "POST",
		},
		"new-visit-options": {
			name:   "new-visit",
			path:   "/new",
			method: "OPTIONS",
		},
		"get-last-get": {
			name:   "get-last",
			path:   "/",
			method: "GET",
		},
		"get-last-options": {
			name:   "get-last",
			path:   "/",
			method: "OPTIONS",
		},

		// TODO: others
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
	browserReqSecret := "beer"
	loginSession := &LoginSession{
		Token:     "tokenAbc123",
		CreatedAt: time.Now(),
		TTL:       0,
	}
	netlogApi := netlog.NewTestApi()

	r := mux.NewRouter()
	handler := NewNetlogHandler(r, netlogApi, browserReqSecret, loginSession)
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
	browserReqSecret := "beer"
	loginSession := &LoginSession{
		Token:     "tokenAbc123",
		CreatedAt: time.Now(),
		TTL:       0,
	}
	netlogApi := netlog.NewTestApi()

	r := mux.NewRouter()
	handler := NewNetlogHandler(r, netlogApi, browserReqSecret, loginSession)
	require.NotNil(t, handler)
	require.NotNil(t, r)

	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	// we remove the auth token:
	//req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	var visits []*netlog.Visit
	err = json.Unmarshal(rr.Body.Bytes(), &visits)
	require.Error(t, err)
	require.Nil(t, visits)
}

func TestNetlogHandler_handleGetAll(t *testing.T) {
	browserReqSecret := "beer"
	loginSession := &LoginSession{
		Token:     "tokenAbc123",
		CreatedAt: time.Now(),
		TTL:       0,
	}
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
	handler := NewNetlogHandler(r, netlogApi, browserReqSecret, loginSession)
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
	assert.Len(t, visits, 2)
}
