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

	var blogPosts []*Blog
	err = json.Unmarshal(rr.Body.Bytes(), &blogPosts)
	require.NoError(t, err)
	require.NotNil(t, blogPosts)

	// check all posts received
	require.Len(t, blogPosts, len(internals.blogApi.posts))
	for i := range blogPosts {
		assert.True(t, blogPosts[i].Id > 0)
		assert.NotEmpty(t, blogPosts[i].Title)
		assert.NotEmpty(t, blogPosts[i].Content)
		assert.False(t, blogPosts[i].CreatedAt.IsZero())
	}
}
