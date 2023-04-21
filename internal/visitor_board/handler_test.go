package visitor_board

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"

	"github.com/go-redis/redismock/v8"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// use TestMain(m *testing.M) { ... } for
// global set-up/tear-down for all the tests in a package
func TestMain(m *testing.M) {
	// Do stuff BEFORE the tests
	m.Run()

	// do stuff AFTER the tests
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
	)
}

func setupVisitorBoardRouterForTests(
	t *testing.T,
	boardClient *Client,
	metricsManager *metrics.Manager,
	browserReqSecret string,
	loginChecker *auth.LoginChecker,
) *mux.Router {
	t.Helper()

	r := mux.NewRouter()
	authMiddleware := middleware.NewAuthMiddlewareHandler(
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

	handler := NewBoardHandler(boardClient, loginChecker)
	handler.SetupRoutes(r)

	return r
}

func TestNewBoardHandler(t *testing.T) {
	r := mux.NewRouter()

	handler := NewBoardHandler(nil, nil)
	handler.SetupRoutes(r)

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
			path:   "/board/messages/delete/{id}",
			method: "DELETE",
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
	boardClient, _, _, _ := getTestBoardClient()
	redisClient, _ := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)
	m := metrics.NewTestManager()
	r := setupVisitorBoardRouterForTests(t, boardClient, m, "", loginChecker)

	req, err := http.NewRequest("GET", "/board/messages/count", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, `{"count":5}`, rr.Body.String())
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
}

func TestBoardHandler_handleGetAllMessages(t *testing.T) {
	boardClient, _, _, initialBoardMessages := getTestBoardClient()
	redisClient, _ := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)
	m := metrics.NewTestManager()
	r := setupVisitorBoardRouterForTests(t, boardClient, m, "", loginChecker)

	req, err := http.NewRequest("GET", "/board/messages/all", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*Message
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)

	// check all messages there
	require.Len(t, boardMessages, len(initialBoardMessages))
	for i := range boardMessages {
		assert.NotNil(t, initialBoardMessages[boardMessages[i].ID])
	}
}

func TestBoardHandler_handleGetLastMessages(t *testing.T) {
	boardClient, _, _, _ := getTestBoardClient()
	redisClient, _ := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)
	m := metrics.NewTestManager()
	r := setupVisitorBoardRouterForTests(t, boardClient, m, "", loginChecker)

	req, err := http.NewRequest("GET", "/board/messages/last/2", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*Message
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)

	// check all messages there
	require.Len(t, boardMessages, 2)
	assert.Equal(t, 4, boardMessages[0].ID)
	assert.Equal(t, 1, boardMessages[1].ID)
}

func TestBoardHandler_handleGetMessagesPage(t *testing.T) {
	boardClient, _, _, initialBoardMessages := getTestBoardClient()
	redisClient, _ := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)
	m := metrics.NewTestManager()
	r := setupVisitorBoardRouterForTests(t, boardClient, m, "", loginChecker)

	req, err := http.NewRequest("GET", "/board/messages/page/2/size/2", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*Message
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
	req, err = http.NewRequest("GET", "/board/messages/page/2/size/200", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)
	require.Len(t, boardMessages, len(initialBoardMessages))

	// invalid arguments
	req, err = http.NewRequest("GET", "/board/messages/page/invalid/size/2", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	assert.Equal(t, "parse form error, parameter <page>\n", rr.Body.String())
}

func TestBoardHandler_handleDeleteMessage(t *testing.T) {
	boardClient, _, _, initialBoardMessages := getTestBoardClient()
	redisClient, redisMock := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)
	m := metrics.NewTestManager()
	r := setupVisitorBoardRouterForTests(t, boardClient, m, "", loginChecker)

	// wrong session token
	req, err := http.NewRequest("DELETE", "/board/messages/delete/2", nil)
	req.Header.Set("Origin", "test")
	req.Header.Set("X-SERJ-TOKEN", "mywrongsecret")
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	messagesCount, err := boardClient.MessagesCount()
	require.NoError(t, err)
	assert.Equal(t, len(initialBoardMessages), messagesCount)

	// session token missing
	req, err = http.NewRequest("DELETE", "/board/messages/delete/2", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	messagesCount, err = boardClient.MessagesCount()
	require.NoError(t, err)
	assert.Equal(t, len(initialBoardMessages), messagesCount)

	// correct secret - messages should get removed
	req, err = http.NewRequest("DELETE", "/board/messages/delete/2", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	redisMock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	newCount, err := boardClient.aeroClient.CountAll()
	require.NoError(t, err)
	assert.Equal(t, "true", rr.Body.String())
	assert.Equal(t, len(initialBoardMessages)-1, newCount)

	// delete same message again - and fail to do so
	req, err = http.NewRequest("DELETE", "/board/messages/delete/2", nil)
	require.NoError(t, err)
	redisMock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("Origin", "test")
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	newCount, err = boardClient.aeroClient.CountAll()
	require.NoError(t, err)
	assert.Equal(t, "false", rr.Body.String())
	assert.Equal(t, len(initialBoardMessages)-1, newCount)

	// delete another one
	req, err = http.NewRequest("DELETE", "/board/messages/delete/3", nil)
	require.NoError(t, err)
	redisMock.ExpectGet("serj-service-session||tokenAbc123").SetVal(fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("Origin", "test")
	req.Header.Set("X-SERJ-TOKEN", "tokenAbc123")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	newCount, err = boardClient.aeroClient.CountAll()
	require.NoError(t, err)
	assert.Equal(t, "true", rr.Body.String())
	assert.Equal(t, len(initialBoardMessages)-2, newCount)

	// get all
	req, err = http.NewRequest("GET", "/board/messages/all", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*Message
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)
	require.Len(t, boardMessages, len(initialBoardMessages)-2)

	for i := range boardMessages {
		// check deleted messages not received
		assert.NotEqual(t, 2, boardMessages[i].ID)
		assert.NotEqual(t, 3, boardMessages[i].ID)
	}
}

func TestBoardHandler_handleNewMessage(t *testing.T) {
	boardClient, _, aeroTestClient, initialBoardMessages := getTestBoardClient()
	redisClient, _ := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)
	m := metrics.NewTestManager()
	r := setupVisitorBoardRouterForTests(t, boardClient, m, "", loginChecker)

	req, err := http.NewRequest("POST", "/board/messages/new", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	req.PostForm = url.Values{}
	req.PostForm.Add("message", "yaba")
	req.PostForm.Add("author", "chris")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "added:5", rr.Body.String())
	messagesCount, err := boardClient.MessagesCount()
	require.NoError(t, err)
	assert.Equal(t, len(initialBoardMessages)+1, messagesCount)
	assert.Equal(t, len(initialBoardMessages)+1, len(aeroTestClient.AeroBinMaps))

	// add new message with empty author
	req, err = http.NewRequest("POST", "/board/messages/new", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	req.PostForm = url.Values{}
	req.PostForm.Add("message", "yaba2")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "added:6", rr.Body.String())
	messagesCount, err = boardClient.MessagesCount()
	require.NoError(t, err)
	assert.Equal(t, len(initialBoardMessages)+2, messagesCount)
	assert.Equal(t, len(initialBoardMessages)+2, len(aeroTestClient.AeroBinMaps))

	// check messages created
	req, err = http.NewRequest("GET", "/board/messages/all", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var boardMessages []*Message
	err = json.Unmarshal(rr.Body.Bytes(), &boardMessages)
	require.NoError(t, err)
	require.NotNil(t, boardMessages)
	require.Equal(t, len(initialBoardMessages)+2, len(boardMessages))

	// check messages are there and came after the previously last one
	lastMsgTime := time.Unix(initialBoardMessages[len(initialBoardMessages)-1].Timestamp, 0)
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

func TestBoardHandler_handleNewMessage_jsonPayload(t *testing.T) {
	boardClient, _, aeroTestClient, initialBoardMessages := getTestBoardClient()
	redisClient, _ := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)
	m := metrics.NewTestManager()
	r := setupVisitorBoardRouterForTests(t, boardClient, m, "", loginChecker)

	newMsgParams := Message{
		Message: "testmsg",
		Author:  "testperson",
	}
	newMsgParamsBytes, err := json.Marshal(newMsgParams)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/board/messages/new", bytes.NewBuffer(newMsgParamsBytes))
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "added:5", rr.Body.String())
	messagesCount, err := boardClient.MessagesCount()
	require.NoError(t, err)
	assert.Equal(t, len(initialBoardMessages)+1, messagesCount)
	assert.Equal(t, len(initialBoardMessages)+1, len(aeroTestClient.AeroBinMaps))

	// with empty message
	newMsgParams = Message{
		Author: "anon",
	}
	newMsgParamsBytes, err = json.Marshal(newMsgParams)
	require.NoError(t, err)

	req, err = http.NewRequest("POST", "/board/messages/new", bytes.NewBuffer(newMsgParamsBytes))
	require.NoError(t, err)
	req.Header.Set("Origin", "test")
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "error, message empty\n", rr.Body.String())
}
