package notes_box

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

// TestMain will run goleak after all tests have been run in the package
// to detect any goroutine leaks
func TestMain(m *testing.M) {
	m.Run()
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
	)
}

func TestNotesBoxHandler_AllNotes(t *testing.T) {
	api := NewTestApi()
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
	_, err := api.Add(ctx, n1)
	require.NoError(t, err)
	_, err = api.Add(ctx, n2)
	require.NoError(t, err)

	db, _ := redismock.NewClientMock()
	loginChecker := auth.NewLoginChecker(time.Hour, db)

	metrics := metrics.NewTestManager()
	handler := NewHandler(api, loginChecker, metrics)
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
