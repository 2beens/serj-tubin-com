package blog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redismock/v8"

	"github.com/go-redis/redis/v8"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlogHandler(t *testing.T) {
	r := mux.NewRouter()
	boardRouter := r.PathPrefix("/blog").Subrouter()

	handler := NewBlogHandler(boardRouter, nil, nil)
	require.NotNil(t, handler)
	require.NotNil(t, boardRouter)

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
		t.Run(caseName, func(t *testing.T) {
			t.Parallel()
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

func getTestBlogApiAndLoginChecker(t *testing.T, redisClient *redis.Client) (*TestApi, *auth.LoginChecker) {
	t.Helper()
	now := time.Now()

	blogApi := NewBlogTestApi()
	for i := 0; i < 5; i++ {
		require.NoError(t, blogApi.AddBlog(&Blog{
			Id:        i,
			Title:     fmt.Sprintf("blog%dtitle", i),
			CreatedAt: now.Add(time.Minute * time.Duration(i)),
			Content:   fmt.Sprintf("blog %d content", i),
		}))
	}

	loginChecker := auth.NewLoginChecker(time.Hour, redisClient)

	return blogApi, loginChecker
}

func TestBlogHandler_handleAll(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

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
	require.Len(t, blogPosts, blogApi.PostsCount())
	for i := range blogPosts {
		assert.True(t, blogPosts[i].Id >= 0)
		assert.NotEmpty(t, blogPosts[i].Title)
		assert.NotEmpty(t, blogPosts[i].Content)
		assert.False(t, blogPosts[i].CreatedAt.IsZero())
	}
}

func TestBlogHandler_handleGetPage(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

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
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

	req, err := http.NewRequest("DELETE", "/blog/delete/3", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	currentPostsCount := blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, blogApi.PostsCount())

	// check that blog was not deleted
	assert.NotNil(t, blogApi.Posts[3])

	// now logged in
	req, err = http.NewRequest("DELETE", "/blog/delete/3", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "deleted:3", rr.Body.String())
	assert.Equal(t, currentPostsCount-1, blogApi.PostsCount())
	assert.Nil(t, blogApi.Posts[3])
}

func TestBlogHandler_handleNewBlog_notLoggedIn(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")
	rr := httptest.NewRecorder()

	currentPostsCount := blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, blogApi.PostsCount())
}

func TestBlogHandler_handleUpdateBlog_notLoggedIn(t *testing.T) {
	redisClient, _ := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")
	rr := httptest.NewRecorder()

	currentPostsCount := blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, blogApi.PostsCount())

	// check that blog was not updated
	assert.Equal(t, "blog4title", blogApi.Posts[4].Title)
}

func TestBlogHandler_handleNewBlog_wrongToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mywrongsecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, blogApi.PostsCount())
}

func TestBlogHandler_handleUpdateBlog_wrongToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "wrongsecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, blogApi.PostsCount())

	// check that blog was not updated
	assert.Equal(t, "blog4title", blogApi.Posts[4].Title)
}

func TestBlogHandler_handleNewBlog_correctToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "added:5", rr.Body.String())
	assert.Equal(t, currentPostsCount+1, blogApi.PostsCount())

	addedPost, ok := blogApi.Posts[5]
	require.True(t, ok)
	require.NotNil(t, addedPost)
	assert.Equal(t, "Nonsense", addedPost.Title)
	assert.Equal(t, "This content makes no sense", addedPost.Content)
	assert.False(t, addedPost.CreatedAt.IsZero())
}

func TestBlogHandler_handleUpdateBlog_correctToken(t *testing.T) {
	redisClient, redisMock := redismock.NewClientMock()
	blogApi, loginChecker := getTestBlogApiAndLoginChecker(t, redisClient)

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), blogApi, loginChecker)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	redisMock.ExpectGet("serj-service-session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "updated:4", rr.Body.String())
	assert.Equal(t, currentPostsCount, blogApi.PostsCount())

	addedPost, ok := blogApi.Posts[4]
	require.True(t, ok)
	require.NotNil(t, addedPost)
	assert.Equal(t, "Nonsense", addedPost.Title)
	assert.Equal(t, "This content makes no sense", addedPost.Content)
	assert.False(t, addedPost.CreatedAt.IsZero())
}
