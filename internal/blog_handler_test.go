package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlogHandler_handleAll(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.loginSession)
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

func TestBlogHandler_handleNewBlog_notLoggedIn(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.loginSession)
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

func TestBlogHandler_handleNewBlog_wrongToken(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.loginSession)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	handler.loginSession.Token = "mywrongsecret"

	rr := httptest.NewRecorder()

	currentPostsCount := internals.blogApi.PostsCount()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	require.Equal(t, "no can do\n", rr.Body.String())
	assert.Equal(t, currentPostsCount, internals.blogApi.PostsCount())
}

func TestBlogHandler_handleNewBlog_correctToken(t *testing.T) {
	internals := newTestingInternals()

	r := mux.NewRouter()
	handler := NewBlogHandler(r.PathPrefix("/blog").Subrouter(), internals.blogApi, internals.loginSession)
	require.NotNil(t, handler)

	req, err := http.NewRequest("POST", "/blog/new", nil)
	require.NoError(t, err)

	req.PostForm = url.Values{}
	req.PostForm.Add("title", "Nonsense")
	req.PostForm.Add("content", "This content makes no sense")

	req.Header.Set("X-SERJ-TOKEN", "mylittlesecret")
	handler.loginSession.Token = "mylittlesecret"

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