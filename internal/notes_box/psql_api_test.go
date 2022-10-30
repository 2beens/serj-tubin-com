package notes_box

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deleteAll(ctx context.Context, psqlApi *PsqlApi) (int64, error) {
	tag, err := psqlApi.db.Exec(ctx, `DELETE FROM note`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func getPsqlApi(t *testing.T) (*PsqlApi, error) {
	t.Helper()

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	t.Logf("using postres host: %s", host)

	return NewPsqlApi(timeoutCtx, host, "5432", "serj_blogs")
}

func TestPsqlApi_BasicCRUD(t *testing.T) {
	api, err := getPsqlApi(t)
	require.NoError(t, err)
	require.NotNil(t, api)
	defer api.CloseDB()

	ctx := context.Background()
	deleted, err := deleteAll(ctx, api)
	t.Logf("test setup, deleted notes: %d", deleted)

	notes, err := api.List(ctx)
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

	addedNote1, err := api.Add(ctx, note1)
	require.NoError(t, err)
	require.NotNil(t, addedNote1)
	addedNote2, err := api.Add(ctx, note2)
	require.NoError(t, err)
	require.NotNil(t, addedNote2)

	assert.Equal(t, note1.Content, addedNote1.Content)
	assert.Equal(t, note1.Title, addedNote1.Title)
	assert.Equal(t, note2.Content, addedNote2.Content)
	assert.Equal(t, note2.Title, addedNote2.Title)

	notes, err = api.List(ctx)
	require.NoError(t, err)
	require.NotNil(t, notes)
	assert.Len(t, notes, 2)

	retrievedNote1, err := api.Get(ctx, addedNote1.Id)
	require.NoError(t, err)
	assert.Equal(t, note1.Content, retrievedNote1.Content)
	assert.Equal(t, note1.Title, retrievedNote1.Title)

	// now remove
	note3 := &Note{
		Title:     "title3",
		CreatedAt: now,
		Content:   "content3",
	}
	addedNote3, err := api.Add(ctx, note3)
	require.NoError(t, err)
	assert.Equal(t, note3.Content, addedNote3.Content)
	assert.Equal(t, note3.Title, addedNote3.Title)

	require.NoError(t, api.Delete(ctx, note3.Id))

	retrievedNote3, err := api.Get(ctx, addedNote3.Id)
	assert.Error(t, err)
	assert.Nil(t, retrievedNote3)
	assert.Contains(t, err.Error(), "note not found")

	require.NoError(t, api.Delete(ctx, note1.Id))
	require.NoError(t, api.Delete(ctx, note2.Id))

	notes, err = api.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, notes)
}
