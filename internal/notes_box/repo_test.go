//go:build integration_test || all_tests

package notes_box

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deleteAll(ctx context.Context, repo *Repo) (int64, error) {
	tag, err := repo.db.Exec(ctx, `DELETE FROM note`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

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

func TestRepo_BasicCRUD(t *testing.T) {
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	ctx := context.Background()
	deleted, err := deleteAll(ctx, repo)
	require.NoError(t, err)
	t.Logf("test setup, deleted notes: %d", deleted)

	notes, err := repo.List(ctx)
	require.NoError(t, err)
	require.Empty(t, notes)

	now := time.Now()
	note1 := &Note{
		Title:     "title1",
		CreatedAt: now,
		Content:   "content1",
	}
	note2 := &Note{
		Title:     "title2",
		CreatedAt: now,
		Content:   "content2",
	}

	addedNote1, err := repo.Add(ctx, note1)
	require.NoError(t, err)
	require.NotNil(t, addedNote1)
	addedNote2, err := repo.Add(ctx, note2)
	require.NoError(t, err)
	require.NotNil(t, addedNote2)

	assert.Equal(t, note1.Content, addedNote1.Content)
	assert.Equal(t, note1.Title, addedNote1.Title)
	assert.Equal(t, note2.Content, addedNote2.Content)
	assert.Equal(t, note2.Title, addedNote2.Title)

	notes, err = repo.List(ctx)
	require.NoError(t, err)
	require.NotNil(t, notes)
	assert.Len(t, notes, 2)

	retrievedNote1, err := repo.Get(ctx, addedNote1.ID)
	require.NoError(t, err)
	assert.Equal(t, note1.Content, retrievedNote1.Content)
	assert.Equal(t, note1.Title, retrievedNote1.Title)

	// now remove
	note3 := &Note{
		Title:     "title3",
		CreatedAt: now,
		Content:   "content3",
	}
	addedNote3, err := repo.Add(ctx, note3)
	require.NoError(t, err)
	assert.Equal(t, note3.Content, addedNote3.Content)
	assert.Equal(t, note3.Title, addedNote3.Title)

	require.NoError(t, repo.Delete(ctx, note3.ID))

	retrievedNote3, err := repo.Get(ctx, addedNote3.ID)
	assert.Error(t, err)
	assert.Nil(t, retrievedNote3)
	assert.Contains(t, err.Error(), "note not found")

	nonExisting, err := repo.Get(ctx, 12341234)
	assert.ErrorIs(t, err, ErrNoteNotFound)
	assert.Nil(t, nonExisting)

	require.NoError(t, repo.Delete(ctx, note1.ID))
	require.NoError(t, repo.Delete(ctx, note2.ID))
	assert.ErrorIs(t, repo.Delete(ctx, 12341234), ErrNoteNotFound)

	notes, err = repo.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, notes)
}

func TestRepo_Update(t *testing.T) {
	repo, shutdown := testRepoSetup(t)
	defer shutdown()

	ctx := context.Background()
	deleted, err := deleteAll(ctx, repo)
	require.NoError(t, err)
	t.Logf("test setup, deleted notes: %d", deleted)

	now := time.Now()
	note1 := &Note{
		Title:     "title1",
		CreatedAt: now,
		Content:   "content1",
	}
	note2 := &Note{
		Title:     "title2",
		CreatedAt: now,
		Content:   "content2",
	}

	addedNote1, err := repo.Add(ctx, note1)
	require.NoError(t, err)
	require.NotNil(t, addedNote1)
	addedNote2, err := repo.Add(ctx, note2)
	require.NoError(t, err)
	require.NotNil(t, addedNote2)

	addedNote1.Content = "new-content"
	require.NoError(t, repo.Update(ctx, addedNote1))
	retrievedNote1, err := repo.Get(ctx, addedNote1.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-content", retrievedNote1.Content)
	assert.Equal(t, note1.Title, retrievedNote1.Title)

	addedNote1.Title = "new-title"
	require.NoError(t, repo.Update(ctx, addedNote1))
	retrievedNote1, err = repo.Get(ctx, addedNote1.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-content", retrievedNote1.Content)
	assert.Equal(t, "new-title", retrievedNote1.Title)

	addedNote1.Title = "new-title-2"
	addedNote1.Content = "new-content-2"
	require.NoError(t, repo.Update(ctx, addedNote1))
	retrievedNote1, err = repo.Get(ctx, addedNote1.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-content-2", retrievedNote1.Content)
	assert.Equal(t, "new-title-2", retrievedNote1.Title)

	retrievedNote2, err := repo.Get(ctx, addedNote2.ID)
	require.NoError(t, err)
	assert.Equal(t, note2.Content, retrievedNote2.Content)
	assert.Equal(t, note2.Title, retrievedNote2.Title)

	addedNote1.Content = ""
	require.Equal(t, "note content empty", repo.Update(ctx, addedNote1).Error())
}
