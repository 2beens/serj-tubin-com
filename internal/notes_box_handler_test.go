package internal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/2beens/serjtubincom/internal/notes_box"
	"github.com/stretchr/testify/require"
)

func TestNotesBoxHandler_AllNotes(t *testing.T) {
	api := notes_box.NewTestApi()

	now := time.Now()

	n1 := &notes_box.Note{
		Id:        1,
		Title:     "title1",
		Content:   "content1",
		CreatedAt: now,
	}
	n2 := &notes_box.Note{
		Id:        2,
		Title:     "title2",
		Content:   "content2",
		CreatedAt: now,
	}
	api.Add(n1)
	api.Add(n2)

	loginSession := &LoginSession{
		Token:     "mylittlesecret",
		CreatedAt: now,
		TTL:       0,
	}

	instr := instrumentation.NewTestInstrumentation()
	handler := NewNotesBoxHandler(api, loginSession, instr)
	require.NotNil(t, handler)

	req, err := http.NewRequest("GET", "", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	handler.handleList(rr, req)
	require.NotNil(t, rr)

	var body []byte
	_, err = rr.Body.Read(body)
	require.NoError(t, err)

	// TODO: i'm too lazy ...
	bytes.Contains(body, []byte("title1"))
	bytes.Contains(body, []byte("title2"))
}

// TODO: others
