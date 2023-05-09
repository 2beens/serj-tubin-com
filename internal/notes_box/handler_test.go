package notes_box

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"

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
		Id:        1,
		Title:     "title1",
		Content:   "content1",
		CreatedAt: now,
	}
	n2 := &Note{
		Id:        2,
		Title:     "title2",
		Content:   "content2",
		CreatedAt: now,
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

	var body []byte
	_, err = rr.Body.Read(body)
	require.NoError(t, err)

	// TODO: i'm too lazy ...
	bytes.Contains(body, []byte("title1"))
	bytes.Contains(body, []byte("title2"))
}

// TODO: others
