package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2beens/serjtubincom/internal/blog"
)

func (s *IntegrationTestSuite) getBlogPostsPage(
	ctx context.Context,
	page int,
	size int,
) blog.PostsResponse {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET", fmt.Sprintf("%s/blog/page/%d/size/%d", serverEndpoint, page, size),
		nil,
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	var postsResponse blog.PostsResponse
	require.NoError(s.T(),
		json.NewDecoder(resp.Body).Decode(&postsResponse),
	)

	return postsResponse
}

func (s *IntegrationTestSuite) deleteBlogPostRequest(
	ctx context.Context,
	authToken string,
	postID int,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE", fmt.Sprintf("%s/blog/delete/%d", serverEndpoint, postID),
		nil,
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-SERJ-TOKEN", authToken)

	return s.httpClient.Do(req)
}

func (s *IntegrationTestSuite) newBlogPostRequest(
	ctx context.Context,
	authToken string,
	post blog.Blog,
) int {
	postJson, err := json.Marshal(post)
	require.NoError(s.T(), err)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST", fmt.Sprintf("%s/blog/new", serverEndpoint),
		bytes.NewReader(postJson),
	)
	require.NoError(s.T(), err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-SERJ-TOKEN", authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), respBytes)

	respParts := bytes.Split(respBytes, []byte(":"))
	require.Equal(s.T(), 2, len(respParts))

	id, err := strconv.Atoi(string(respParts[1]))
	require.NoError(s.T(), err)

	return id
}

func (s *IntegrationTestSuite) TestBlogs() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.T().Run("try add blog without auth token", func(t *testing.T) {
		postJson, err := json.Marshal(blog.Blog{
			Title:   "test blog",
			Content: "test content",
		})
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(
			ctx,
			"POST", fmt.Sprintf("%s/blog/new", serverEndpoint),
			bytes.NewReader(postJson),
		)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		require.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	s.T().Run("add posts and try delete", func(t *testing.T) {
		authToken := s.doLogin(ctx)

		now := time.Now()
		blogPost1 := blog.Blog{
			Title:     "test blog 1",
			Content:   "test content 1",
			CreatedAt: now.Add(-time.Minute),
		}
		postID1 := s.newBlogPostRequest(ctx, authToken, blogPost1)
		require.NotZero(t, postID1)

		blogPost2 := blog.Blog{
			Title:     "test blog 2",
			Content:   "test content 2",
			CreatedAt: now,
		}
		postID2 := s.newBlogPostRequest(ctx, authToken, blogPost2)
		require.NotZero(t, postID2)

		blogsPage := s.getBlogPostsPage(ctx, 1, 10)
		require.Equal(t, 2, len(blogsPage.Posts))
		require.Equal(t, 2, blogsPage.Total)

		require.Equal(t, postID2, blogsPage.Posts[0].ID)
		require.Equal(t, blogPost2.Title, blogsPage.Posts[0].Title)
		require.Equal(t, blogPost2.Content, blogsPage.Posts[0].Content)
		require.NotZero(t, blogsPage.Posts[0].CreatedAt)
		require.Equal(t, postID1, blogsPage.Posts[1].ID)
		require.Equal(t, blogPost1.Title, blogsPage.Posts[1].Title)
		require.Equal(t, blogPost1.Content, blogsPage.Posts[1].Content)
		require.NotZero(t, blogsPage.Posts[1].CreatedAt)

		// try delete with invalid token
		resp, err := s.deleteBlogPostRequest(ctx, "invalid-token", postID1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		// try delete with valid token
		resp, err = s.deleteBlogPostRequest(ctx, authToken, postID1)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		blogsPage = s.getBlogPostsPage(ctx, 1, 10)
		require.Equal(t, 1, len(blogsPage.Posts))
		require.Equal(t, 1, blogsPage.Total)
	})
}
