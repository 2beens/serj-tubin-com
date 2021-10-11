package notes_box

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPsqlApi_BasicCRUD(t *testing.T) {
	api, err := NewPsqlApi("localhost", "5432", "testing")
	require.NoError(t, err)
	require.NotNil(t, api)

	defer api.CloseDB()

	notes, err := api.List()
	require.NoError(t, err)
	originalLen := len(notes)

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

	addedNote1, err := api.Add(note1)
	require.NoError(t, err)
	require.NotNil(t, addedNote1)
	defer api.Remove(addedNote1.Id)
	addedNote2, err := api.Add(note2)
	require.NoError(t, err)
	require.NotNil(t, addedNote2)
	defer api.Remove(addedNote2.Id)

	assert.Equal(t, note1.Content, addedNote1.Content)
	assert.Equal(t, note1.Title, addedNote1.Title)
	assert.Equal(t, note2.Content, addedNote2.Content)
	assert.Equal(t, note2.Title, addedNote2.Title)

	notes, err = api.List()
	require.NoError(t, err)
	require.NotNil(t, notes)
	assert.Len(t, notes, originalLen+2)

	retrievedNote1, err := api.Get(addedNote1.Id)
	require.NoError(t, err)
	assert.Equal(t, note1.Content, retrievedNote1.Content)
	assert.Equal(t, note1.Title, retrievedNote1.Title)

	// now remove
	note3 := &Note{
		Title:     "title3",
		CreatedAt: now,
		Content:   "content3",
	}
	addedNote3, err := api.Add(note3)
	assert.Equal(t, note3.Content, addedNote3.Content)
	assert.Equal(t, note3.Title, addedNote3.Title)

	removed, err := api.Remove(note3.Id)
	require.NoError(t, err)
	assert.True(t, removed)

	retrievedNote3, err := api.Get(addedNote3.Id)
	assert.Error(t, err)
	assert.Nil(t, retrievedNote3)
	assert.Contains(t, err.Error(), "failed to get note")
}
