//go:build integration_test || all_tests

package blog

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	gofakeit "github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getPsqlApi(t *testing.T) (*PsqlApi, error) {
	t.Helper()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	t.Logf("using postres host: %s", host)

	return NewBlogPsqlApi(timeoutCtx, host, "5432", "serj_blogs")
}

func TestNewBlogPsqlApi(t *testing.T) {
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)
	require.NotNil(t, psqlApi)
	assert.NotNil(t, psqlApi.db)
}

func TestPsqlApi_AddBlog_DeleteBlog(t *testing.T) {
	ctx := context.Background()
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)

	blogsCount, err := psqlApi.BlogsCount(ctx)
	require.NoError(t, err)

	now := time.Now().Add(-time.Minute)

	b1 := &Blog{
		Title:   "b1",
		Content: "content1",
	}
	err = psqlApi.AddBlog(ctx, b1)
	require.NoError(t, err)
	b2 := &Blog{
		Title:   "b2",
		Content: "content2",
	}
	err = psqlApi.AddBlog(ctx, b2)
	require.NoError(t, err)
	b3 := &Blog{
		Title:   "b3",
		Content: "content3",
	}
	err = psqlApi.AddBlog(ctx, b3)
	require.NoError(t, err)

	assert.NotEqual(t, b1.Id, b2.Id)
	assert.NotEqual(t, b1.Id, b3.Id)
	assert.NotEqual(t, b2.Id, b3.Id)
	assert.True(t, now.Before(b1.CreatedAt), "%v should be before %v", now, b1.CreatedAt)
	assert.True(t, now.Before(b2.CreatedAt), "%v should be before %v", now, b2.CreatedAt)
	assert.True(t, now.Before(b2.CreatedAt), "%v should be before %v", now, b3.CreatedAt)

	blogsCountAfter, err := psqlApi.BlogsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3+blogsCount, blogsCountAfter)

	// now delete b2
	assert.ErrorIs(t, psqlApi.DeleteBlog(ctx, 25342523), ErrBlogNotFound)
	require.NoError(t, psqlApi.DeleteBlog(ctx, b2.Id))
	_, err = psqlApi.GetBlog(ctx, b2.Id)
	assert.ErrorIs(t, err, ErrBlogNotFound)
}

func TestPsqlApi_UpdateBlog_BlogClapped(t *testing.T) {
	ctx := context.Background()
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)

	clapsCount := 10
	blog := &Blog{
		Title:   gofakeit.Name(),
		Content: gofakeit.Address().Address,
		Claps:   clapsCount,
	}
	err = psqlApi.AddBlog(ctx, blog)
	require.NoError(t, err)

	require.NoError(t, psqlApi.UpdateBlog(ctx, blog.Id, "newtitle", "newcontent"))

	updatedBlog, err := psqlApi.GetBlog(ctx, blog.Id)
	require.NoError(t, err)
	require.NotNil(t, updatedBlog)
	assert.Equal(t, "newcontent", updatedBlog.Content)
	assert.Equal(t, "newtitle", updatedBlog.Title)
	assert.Equal(t, clapsCount, updatedBlog.Claps)

	// assert claps
	assert.ErrorIs(t, psqlApi.BlogClapped(ctx, 25342523), ErrBlogNotFound)
	require.NoError(t, psqlApi.BlogClapped(ctx, blog.Id))
	require.NoError(t, psqlApi.BlogClapped(ctx, blog.Id))
	require.NoError(t, psqlApi.BlogClapped(ctx, blog.Id))

	updatedBlog, err = psqlApi.GetBlog(ctx, blog.Id)
	require.NoError(t, err)
	require.NotNil(t, updatedBlog)
	assert.Equal(t, "newcontent", updatedBlog.Content)
	assert.Equal(t, "newtitle", updatedBlog.Title)
	assert.Equal(t, clapsCount+3, updatedBlog.Claps)
}

func TestPsqlApi_All(t *testing.T) {
	ctx := context.Background()
	psqlApi, err := getPsqlApi(t)
	require.NoError(t, err)

	blogsCount, err := psqlApi.BlogsCount(ctx)
	require.NoError(t, err)

	addedCount := 5
	for i := 1; i <= addedCount; i++ {
		b := &Blog{
			Title:   fmt.Sprintf("b %d", i),
			Content: fmt.Sprintf("content %d", i),
		}
		err = psqlApi.AddBlog(ctx, b)
		require.NoError(t, err)
	}

	blogsCountAfter, err := psqlApi.BlogsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, addedCount+blogsCount, blogsCountAfter)

	allBlogs, err := psqlApi.All(ctx)
	require.NoError(t, err)
	assert.True(t, len(allBlogs) >= addedCount)
}
