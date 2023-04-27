//go:build integration_test || all_tests

package blog

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/db"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRepoSetup(t *testing.T) (*Repo, func()) {
	t.Helper()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	t.Logf("using postres host: %s", host)

	dbPool, err := db.NewDBPool(timeoutCtx, db.NewDBPoolParams{
		DBHost:         host,
		DBPort:         "5432",
		DBName:         "serj_blogs",
		TracingEnabled: false,
	})
	require.NoError(t, err)

	return NewRepo(dbPool), func() {
		dbPool.Close()
	}
}

func TestRepo_AddBlog_DeleteBlog(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	blogsCount, err := repo.BlogsCount(ctx)
	require.NoError(t, err)

	now := time.Now().Add(-time.Minute)

	b1 := &Blog{
		Title:   "b1",
		Content: "content1",
	}
	err = repo.AddBlog(ctx, b1)
	require.NoError(t, err)
	b2 := &Blog{
		Title:   "b2",
		Content: "content2",
	}
	err = repo.AddBlog(ctx, b2)
	require.NoError(t, err)
	b3 := &Blog{
		Title:   "b3",
		Content: "content3",
	}
	err = repo.AddBlog(ctx, b3)
	require.NoError(t, err)

	assert.NotEqual(t, b1.ID, b2.ID)
	assert.NotEqual(t, b1.ID, b3.ID)
	assert.NotEqual(t, b2.ID, b3.ID)
	assert.True(t, now.Before(b1.CreatedAt), "%v should be before %v", now, b1.CreatedAt)
	assert.True(t, now.Before(b2.CreatedAt), "%v should be before %v", now, b2.CreatedAt)
	assert.True(t, now.Before(b2.CreatedAt), "%v should be before %v", now, b3.CreatedAt)

	blogsCountAfter, err := repo.BlogsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3+blogsCount, blogsCountAfter)

	// now delete b2
	assert.ErrorIs(t, repo.DeleteBlog(ctx, 25342523), ErrBlogNotFound)
	require.NoError(t, repo.DeleteBlog(ctx, b2.ID))
	_, err = repo.GetBlog(ctx, b2.ID)
	assert.ErrorIs(t, err, ErrBlogNotFound)
}

func TestRepo_UpdateBlog_BlogClapped(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	clapsCount := 10
	blog := &Blog{
		Title:   gofakeit.Name(),
		Content: gofakeit.Address().Address,
		Claps:   clapsCount,
	}
	err := repo.AddBlog(ctx, blog)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateBlog(ctx, blog.ID, "newtitle", "newcontent"))

	updatedBlog, err := repo.GetBlog(ctx, blog.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedBlog)
	assert.Equal(t, "newcontent", updatedBlog.Content)
	assert.Equal(t, "newtitle", updatedBlog.Title)
	assert.Equal(t, clapsCount, updatedBlog.Claps)

	// assert claps
	assert.ErrorIs(t, repo.BlogClapped(ctx, 25342523), ErrBlogNotFound)
	require.NoError(t, repo.BlogClapped(ctx, blog.ID))
	require.NoError(t, repo.BlogClapped(ctx, blog.ID))
	require.NoError(t, repo.BlogClapped(ctx, blog.ID))

	updatedBlog, err = repo.GetBlog(ctx, blog.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedBlog)
	assert.Equal(t, "newcontent", updatedBlog.Content)
	assert.Equal(t, "newtitle", updatedBlog.Title)
	assert.Equal(t, clapsCount+3, updatedBlog.Claps)
}

func TestRepo_All(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	blogsCount, err := repo.BlogsCount(ctx)
	require.NoError(t, err)

	addedCount := 5
	for i := 1; i <= addedCount; i++ {
		b := &Blog{
			Title:   fmt.Sprintf("b %d", i),
			Content: fmt.Sprintf("content %d", i),
		}
		err = repo.AddBlog(ctx, b)
		require.NoError(t, err)
	}

	blogsCountAfter, err := repo.BlogsCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, addedCount+blogsCount, blogsCountAfter)

	allBlogs, err := repo.All(ctx)
	require.NoError(t, err)
	assert.True(t, len(allBlogs) >= addedCount)
}

func TestRepo_GetBlogsPage(t *testing.T) {
	ctx := context.Background()
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	addedCount := 5
	for i := 1; i <= addedCount; i++ {
		b := &Blog{
			Title:   fmt.Sprintf("b %d", i),
			Content: fmt.Sprintf("content %d", i),
		}
		require.NoError(t, repo.AddBlog(ctx, b))
	}

	blogs, err := repo.GetBlogsPage(ctx, 2, 2)
	require.NoError(t, err)
	assert.Len(t, blogs, 2)

	blogs, err = repo.GetBlogsPage(ctx, 1, 1)
	require.NoError(t, err)
	assert.Len(t, blogs, 1)

	blogs, err = repo.GetBlogsPage(ctx, 1, addedCount)
	require.NoError(t, err)
	assert.Len(t, blogs, addedCount)
}
