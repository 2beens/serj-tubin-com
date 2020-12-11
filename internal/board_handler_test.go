package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

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
			route := r.Get(route.name)
			require.NotNil(t, route)
			isMatch := route.Match(req, routeMatch)
			assert.True(t, isMatch, caseName)
		})
	}
}

func TestBoardHandler_handleMessagesCount(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBoardHandler(r, internals.board, "secret")
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "/messages/count", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
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
	var found1, found2 bool
	for i := range boardMessages {
		if boardMessages[i].ID == 2 {
			found1 = true
		}
		if boardMessages[i].ID == 3 {
			found2 = true
		}
	}
	assert.True(t, found1)
	assert.True(t, found2)

	// big size
	req, err = http.NewRequest("GET", "/messages/page/2/size/200", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)
	require.Len(t, boardMessages, len(internals.initialBoardMessages))

	// invalid arguments
	req, err = http.NewRequest("GET", "/messages/page/invalid/size/2", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.Equal(t, "parse form error, parameter <page>\n", rr.Body.String())
}

func TestBoardHandler_handleDeleteMessage(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBoardHandler(r, internals.board, "secret-word")
	require.NotNil(t, handler)

	// wrong secret
	req, err := http.NewRequest("GET", "/messages/delete/2/secret-word-blabla", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Equal(t, len(internals.initialBoardMessages), internals.board.messagesCounter)

	// correct secret - messages should get removed
	req, err = http.NewRequest("GET", "/messages/delete/2/secret-word", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	newCount, err := internals.board.aeroClient.CountAll()
	require.NoError(t, err)
	assert.Equal(t, "true", rr.Body.String())
	assert.Equal(t, len(internals.initialBoardMessages)-1, newCount)

	// delete same message again - and fail to do so
	req, err = http.NewRequest("GET", "/messages/delete/2/secret-word", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	newCount, err = internals.board.aeroClient.CountAll()
	require.NoError(t, err)
	assert.Equal(t, "false", rr.Body.String())
	assert.Equal(t, len(internals.initialBoardMessages)-1, newCount)

	// delete another one
	req, err = http.NewRequest("GET", "/messages/delete/3/secret-word", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	newCount, err = internals.board.aeroClient.CountAll()
	require.NoError(t, err)
	assert.Equal(t, "true", rr.Body.String())
	assert.Equal(t, len(internals.initialBoardMessages)-2, newCount)

	// get all
	req, err = http.NewRequest("GET", "/messages/all", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*BoardMessage
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)
	require.Len(t, boardMessages, len(internals.initialBoardMessages)-2)

	for i := range boardMessages {
		// check deleted messages not received
		assert.NotEqual(t, 2, boardMessages[i].ID)
		assert.NotEqual(t, 3, boardMessages[i].ID)
	}
}

func TestBoardHandler_handleMessagesRange(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBoardHandler(r, internals.board, "secret")
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "/messages/from/1/to/3", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*BoardMessage
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)

	// order not guaranteed
	require.Len(t, boardMessages, 3)
	var found1, found2, found3 bool
	for i := range boardMessages {
		if boardMessages[i].ID == 1 {
			found1 = true
		}
		if boardMessages[i].ID == 2 {
			found2 = true
		}
		if boardMessages[i].ID == 3 {
			found3 = true
		}
	}
	assert.True(t, found1)
	assert.True(t, found2)
	assert.True(t, found3)
}

func TestBoardHandler_handleNewMessage(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBoardHandler(r, internals.board, "secret")
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/messages/new", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("message", "yaba")
	req.PostForm.Add("author", "chris")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "added", rr.Body.String())
	assert.Equal(t, len(internals.initialBoardMessages)+1, internals.board.messagesCounter)
	assert.Equal(t, len(internals.initialBoardMessages)+1, len(internals.aeroTestClient.AeroBinMaps))

	// add new message with empty author
	req, err = http.NewRequest("POST", "/messages/new", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("message", "yaba2")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "added", rr.Body.String())
	assert.Equal(t, len(internals.initialBoardMessages)+2, internals.board.messagesCounter)
	assert.Equal(t, len(internals.initialBoardMessages)+2, len(internals.aeroTestClient.AeroBinMaps))

	// check messages created
	req, err = http.NewRequest("GET", "/messages/all", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*BoardMessage
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)
	require.Equal(t, len(internals.initialBoardMessages)+2, len(boardMessages))

	// check messages are there and came after the previously last one
	lastMsgTime := time.Unix(internals.lastInitialMessage.Timestamp, 0)
	var firstFound, secondFound bool
	for i := range boardMessages {
		msgTime := time.Unix(boardMessages[i].Timestamp, 0)
		if boardMessages[i].Message == "yaba" && boardMessages[i].Author == "chris" {
			if msgTime.After(lastMsgTime) || msgTime.Equal(lastMsgTime) {
				firstFound = true
			}
		}
		if boardMessages[i].Message == "yaba2" && boardMessages[i].Author == "anon" {
			if msgTime.After(lastMsgTime) || msgTime.Equal(lastMsgTime) {
				secondFound = true
			}
		}
	}
	assert.True(t, firstFound)
	assert.True(t, secondFound)
}
