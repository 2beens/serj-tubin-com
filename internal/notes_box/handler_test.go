package notes_box

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// TestMain will run goleak after all tests have been run in the package
// to detect any goroutine leaks
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
	)
}

func TestNotesBoxHandler_AllNotes(t *testing.T) {
	repo := newRepoMock()
	now := time.Now()
	n1 := &Note{
		ID:        1,
		Title:     "title1",
		Content:   "content1",
		CreatedAt: now.Add(-time.Hour),
	}
	n2 := &Note{
		ID:        2,
		Title:     "title2",
		Content:   "content2",
		CreatedAt: now.Add(-time.Hour * 2),
	}

	ctx := context.Background()
	_, err := repo.Add(ctx, n1)
	require.NoError(t, err)
	_, err = repo.Add(ctx, n2)
	require.NoError(t, err)

	metrics := metrics.NewTestManager()
	handler := NewHandler(repo, metrics)
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	handler.HandleList(rr, req)
	require.NotNil(t, rr)

	var notesListRes NotesListResponse
	err = json.Unmarshal(rr.Body.Bytes(), &notesListRes)
	require.NoError(t, err)

	require.Len(t, notesListRes.Notes, 2)
	assert.Equal(t, n1.ID, notesListRes.Notes[0].ID)
	assert.Equal(t, n1.Title, notesListRes.Notes[0].Title)
	assert.Equal(t, n1.Content, notesListRes.Notes[0].Content)
	assert.Equal(t, n2.ID, notesListRes.Notes[1].ID)
	assert.Equal(t, n2.Title, notesListRes.Notes[1].Title)
	assert.Equal(t, n2.Content, notesListRes.Notes[1].Content)
}

// TODO: others
