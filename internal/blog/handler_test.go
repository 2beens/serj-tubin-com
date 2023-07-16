package blog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/middleware"
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

func setupBlogRouterForTests(t *testing.T, repo *repoMock, loginChecker *auth.LoginChecker) *mux.Router {
	t.Helper()

	r := mux.NewRouter()
	authMiddleware := middleware.NewAuthMiddlewareHandler(
		"n/a",
		"browserRequestsSecret",
		loginChecker,
	)
	r.Use(authMiddleware.AuthCheck())

	NewBlogHandler(repo, loginChecker).SetupRoutes(r)

	return r
}

func getRepoMockAndLoginChecker(t *testing.T, redisClient *redis.Client) (*repoMock, *auth.LoginChecker) {
	t.Helper()
	now := time.Now()

	repoMock := newRepoMock()
	for i := 0; i < 5; i++ {
		require.NoError(t, repoMock.AddBlog(context.Background(), &Blog{
			ID:        i,
			Title:     fmt.Sprintf("blog%dtitle", i),
			CreatedAt: now.Add(time.Minute * time.Duration(i)),
			Content:   fmt.Sprintf("blog %d content", i),
		}))
	}

	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)

	return repoMock, loginChecker
}

func TestNewBlogHandler(t *testing.T) {
	r := mux.NewRouter()

	handler := NewBlogHandler(nil, nil)
	handler.SetupRoutes(r)

	for caseName, route := range map[string]struct {
		name   string
		path   string
		method string
	}{
		"new-blog-post": {
			name:   "new-blog",
			path:   "/blog/new",
			method: "POST",
		},
		"new-blog-options": {
			name:   "new-blog",
			path:   "/blog/new",
			method: "OPTIONS",
		},
		"update-blog-post": {
			name:   "update-blog",
			path:   "/blog/update",
			method: "POST",
		},
		"update-blog-options": {
			name:   "update-blog",
			path:   "/blog/update",
			method: "OPTIONS",
		},
		"delete-blog-post": {
			name:   "delete-blog",
			path:   "/blog/delete/1",
			method: "DELETE",
		},
		"delete-blog-options": {
			name:   "delete-blog",
			path:   "/blog/delete/1",
			method: "OPTIONS",
		},
		"all-blog-post": {
			name:   "all-blogs",
			path:   "/blog/all",
			method: "GET",
		},
		"blog-posts-page": {
			name:   "blogs-page",
			path:   "/blog/page/1/size/2",
			method: "GET",
		},
	} {
		nextRoute := route
		cn := caseName
		t.Run(cn, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequest(nextRoute.method, nextRoute.path, nil)
			require.NoError(t, err)

			routeMatch := &mux.RouteMatch{}
			gotRoute := r.Get(nextRoute.name)
			require.NotNil(t, gotRoute)
			isMatch := gotRoute.Match(req, routeMatch)
			assert.True(t, isMatch, cn)
		})
	}
}

func TestBlogHandler_handleAll(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("GET", "/blog/all", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var blogPosts []*Blog
	err = json.Unmarshal(rr.Body.Bytes(), &blogPosts)
	require.NoError(t, err)
	require.NotNil(t, blogPosts)

	// check all posts received
	require.Len(t, blogPosts, repoMock.PostsCount())
	for i := range blogPosts {
		assert.True(t, blogPosts[i].ID >= 0)
		assert.NotEmpty(t, blogPosts[i].Title)
		assert.NotEmpty(t, blogPosts[i].Content)
		assert.False(t, blogPosts[i].CreatedAt.IsZero())
	}
}

func TestBlogHandler_handleGetPage(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("GET", "/blog/page/2/size/2", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	receivedJson := rr.Body.String()
	assert.True(t, strings.Contains(receivedJson, "blog2title"))
	assert.True(t, strings.Contains(receivedJson, "blog3title"))
}

func TestBlogHandler_handleDelete(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	handler := NewBlogHandler(repoMock, loginChecker)
	handler.SetupRoutes(r.PathPrefix("/blog").Subrouter())

	req, err := http.NewRequest("DELETE", "/blog/delete/3", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())

	// check that blog was not deleted
	assert.NotNil(t, repoMock.Posts[3])

	// now logged in
	req, err = http.NewRequest("DELETE", "/blog/delete/3", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "deleted:3", rr.Body.String())
	assert.Equal(t, currentPostsCount-1, repoMock.PostsCount())
	assert.Nil(t, repoMock.Posts[3])
}

func TestBlogHandler_handleNewBlog_notLoggedIn(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	handler := NewBlogHandler(repoMock, loginChecker)
	handler.SetupRoutes(r.PathPrefix("/blog").Subrouter())

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")
	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())
}

func TestBlogHandler_handleUpdateBlog_notLoggedIn(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")
	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())

	// check that blog was not updated
	assert.Equal(t, "blog4title", repoMock.Posts[4].Title)
}

func TestBlogHandler_handleNewBlog_wrongToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mywrongsecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())
}

func TestBlogHandler_handleUpdateBlog_wrongToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "wrongsecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())

	// check that blog was not updated
	assert.Equal(t, "blog4title", repoMock.Posts[4].Title)
}

func TestBlogHandler_handleNewBlog_correctToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "added:5", rr.Body.String())
	assert.Equal(t, currentPostsCount+1, repoMock.PostsCount())

	addedPost, ok := repoMock.Posts[5]
	require.True(t, ok)
	require.NotNil(t, addedPost)
	assert.Equal(t, "Nonsense", addedPost.Title)
	assert.Equal(t, "This content makes no sense", addedPost.Content)
	assert.False(t, addedPost.CreatedAt.IsZero())
}

func TestBlogHandler_handleNewBlog_jsonPayload_correctToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	handler := NewBlogHandler(repoMock, loginChecker)
	handler.SetupRoutes(r.PathPrefix("/blog").Subrouter())

	newBlogParams := newBlogRequest{
		Title:   "test title",
		Content: "test content",
	}
	newBlogParamsBytes, err := json.Marshal(newBlogParams)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/blog/new", bytes.NewBuffer(newBlogParamsBytes))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusCreated, rr.Code)
	require.Equal(t, "added:5", rr.Body.String())
	assert.Equal(t, currentPostsCount+1, repoMock.PostsCount())

	addedPost, ok := repoMock.Posts[5]
	require.True(t, ok)
	require.NotNil(t, addedPost)
	assert.Equal(t, newBlogParams.Title, addedPost.Title)
	assert.Equal(t, newBlogParams.Content, addedPost.Content)
	assert.False(t, addedPost.CreatedAt.IsZero())
}

func TestBlogHandler_handleUpdateBlog_correctToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "updated:4", rr.Body.String())
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())

	addedPost, ok := repoMock.Posts[4]
	require.True(t, ok)
	require.NotNil(t, addedPost)
	assert.Equal(t, "Nonsense", addedPost.Title)
	assert.Equal(t, "This content makes no sense", addedPost.Content)
	assert.False(t, addedPost.CreatedAt.IsZero())
}

func TestBlogHandler_handleBlogClapped_correctToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	repoMock, loginChecker := getRepoMockAndLoginChecker(t, redisClient)
	r := setupBlogRouterForTests(t, repoMock, loginChecker)

	req, err := http.NewRequest("PATCH", "/blog/clap", nil)
	require.NoError(t, err)

	blog0 := repoMock.Posts[0]
	assert.Equal(t, 0, blog0.Claps)

	req.PostForm = url.Values{}
	req.PostForm.Add("id", fmt.Sprintf("%d", blog0.ID))
	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))
	rr := httptest.NewRecorder()

	currentPostsCount := repoMock.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, fmt.Sprintf("updated:%d", blog0.ID), rr.Body.String())
	assert.Equal(t, 1, repoMock.Posts[blog0.ID].Claps)
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())

	req, err = http.NewRequest("PATCH", "/blog/clap", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("id", fmt.Sprintf("%d", blog0.ID))
	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))
	rr = httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, fmt.Sprintf("updated:%d", blog0.ID), rr.Body.String())
	assert.Equal(t, 2, repoMock.Posts[blog0.ID].Claps)
	assert.Equal(t, currentPostsCount, repoMock.PostsCount())
}
