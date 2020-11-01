package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBoardHandler(t *testing.T) {
	r := mux.NewRouter()
	boardRouter := r.PathPrefix("/board").Subrouter()

	handler := NewBoardHandler(boardRouter, nil, "secret")
	require.NotNil(t, handler)
	require.NotNil(t, boardRouter)

	for caseName, route := range map[string]struct {
		name   string
		path   string
		method string
	}{
		"new-message-post": {
			name:   "new-message",
			path:   "/board/messages/new",
			method: "POST",
		},
		"new-message-options": {
			name:   "new-message",
			path:   "/board/messages/new",
			method: "POST",
		},
		"delete-message": {
			name:   "delete-message",
			path:   "/board/messages/delete/{id}/{secret}",
			method: "GET",
		},
		"count-messages": {
			name:   "count-messages",
			path:   "/board/messages/count",
			method: "GET",
		},
		"all-messages": {
			name:   "all-messages",
			path:   "/board/messages/all",
			method: "GET",
		},
		"last-messages": {
			name:   "last-messages",
			path:   "/board/messages/last/{limit}",
			method: "GET",
		},
		"messages-range": {
			name:   "messages-range",
			path:   "/board/messages/from/{from}/to/{to}",
			method: "GET",
		},
		"messages-page": {
			name:   "messages-page",
			path:   "/board/messages/page/{page}/size/{size}",
			method: "GET",
		},
	} {
		t.Run(caseName, func(t *testing.T) {
			req, err := http.NewRequest(route.method, route.path, nil)
			require.NoError(t, err)

			routeMatch := &mux.RouteMatch{}
			isMatch := r.Get(route.name).Match(req, routeMatch)
			assert.True(t, isMatch, caseName)
		})
	}
}

func TestBoardHandler_handleMessagesCount(t *testing.T) {
	internals := newTestingInternals()

	handler := NewBoardHandler(mux.NewRouter(), internals.board, "secret")
	require.NotNil(t, handler)

	req, err := http.NewRequest("-", "-", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	handler.handleMessagesCount(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, `{"count":5}`, rr.Body.String())
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestBoardHandler_handleGetAllMessages(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBoardHandler(r, internals.board, "secret")
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "/messages/all", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*BoardMessage
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)

	// check all messages there
	require.Len(t, boardMessages, len(internals.initialBoardMessages))
	for i := range boardMessages {
		assert.NotNil(t, internals.initialBoardMessages[boardMessages[i].ID])
	}
}

func TestBoardHandler_handleGetLastMessages(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBoardHandler(r, internals.board, "secret")
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "/messages/last/2", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*BoardMessage
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)

	// check all messages there
	require.Len(t, boardMessages, 2)
	assert.Equal(t, 4, boardMessages[0].ID)
	assert.Equal(t, 1, boardMessages[1].ID)
}

func TestBoardHandler_handleGetMessagesPage(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBoardHandler(r, internals.board, "secret")
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "/messages/page/2/size/2", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*BoardMessage
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)

	require.Len(t, boardMessages, 2)
	assert.Equal(t, 2, boardMessages[0].ID)
	assert.Equal(t, 3, boardMessages[1].ID)
}
