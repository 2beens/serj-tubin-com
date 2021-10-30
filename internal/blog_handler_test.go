package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/blog"
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

func TestBlogHandler_handleAll(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "/blog/all", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var blogPosts []*blog.Blog
	err = json.Unmarshal(rr.Body.Bytes(), &blogPosts)
	require.NoError(t, err)
	require.NotNil(t, blogPosts)

	// check all posts received
	require.Len(t, blogPosts, internals.blogApi.PostsCount())
	for i := range blogPosts {
		assert.True(t, blogPosts[i].Id >= 0)
		assert.NotEmpty(t, blogPosts[i].Title)
		assert.NotEmpty(t, blogPosts[i].Content)
		assert.False(t, blogPosts[i].CreatedAt.IsZero())
	}
}

func TestBlogHandler_handleGetPage(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
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
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("DELETE", "/blog/delete/3", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, internals.blogApi.PostsCount())

	// check that blog was not deleted
	assert.NotNil(t, internals.blogApi.Posts[3])

	// now logged in
	req, err = http.NewRequest("DELETE", "/blog/delete/3", nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	internals.redisMock.ExpectGet("session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "deleted:3", rr.Body.String())
	assert.Equal(t, currentPostsCount-1, internals.blogApi.PostsCount())
	assert.Nil(t, internals.blogApi.Posts[3])
}

func TestBlogHandler_handleNewBlog_notLoggedIn(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")
	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, internals.blogApi.PostsCount())
}

func TestBlogHandler_handleUpdateBlog_notLoggedIn(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")
	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, internals.blogApi.PostsCount())

	// check that blog was not updated
	assert.Equal(t, "blog4title", internals.blogApi.Posts[4].Title)
}

func TestBlogHandler_handleNewBlog_wrongToken(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mywrongsecret")
	internals.redisMock.ExpectGet("session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, internals.blogApi.PostsCount())
}

func TestBlogHandler_handleUpdateBlog_wrongToken(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "wrongsecret")
	internals.redisMock.ExpectGet("session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, internals.blogApi.PostsCount())

	// check that blog was not updated
	assert.Equal(t, "blog4title", internals.blogApi.Posts[4].Title)
}

func TestBlogHandler_handleNewBlog_correctToken(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	internals.redisMock.ExpectGet("session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "added:5", rr.Body.String())
	assert.Equal(t, currentPostsCount+1, internals.blogApi.PostsCount())

	addedPost, ok := internals.blogApi.Posts[5]
	require.True(t, ok)
	require.NotNil(t, addedPost)
	assert.Equal(t, "Nonsense", addedPost.Title)
	assert.Equal(t, "This content makes no sense", addedPost.Content)
	assert.False(t, addedPost.CreatedAt.IsZero())
}

func TestBlogHandler_handleUpdateBlog_correctToken(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.authService)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/update", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("id", "4")
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	internals.redisMock.ExpectGet("session||mylittlesecret").SetVal(fmt.Sprintf("%d", time.Now().Unix()))

	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "updated:4", rr.Body.String())
	assert.Equal(t, currentPostsCount, internals.blogApi.PostsCount())

	addedPost, ok := internals.blogApi.Posts[4]
	require.True(t, ok)
	require.NotNil(t, addedPost)
	assert.Equal(t, "Nonsense", addedPost.Title)
	assert.Equal(t, "This content makes no sense", addedPost.Content)
	assert.False(t, addedPost.CreatedAt.IsZero())
}
