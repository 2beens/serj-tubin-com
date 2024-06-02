package test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	notesBox "github.com/2beens/serjtubincom/internal/notes_box"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) TestNotesBox() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t := s.T()
	token := s.doLogin(ctx)

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/notes", serverEndpoint), nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-SERJ-TOKEN", token)

	resp, err := s.httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var notesResp notesBox.NotesListResponse
	require.NoError(t, json.Unmarshal(respBytes, &notesResp))
	assert.Empty(t, notesResp.Notes)
	assert.Equal(t, 0, notesResp.Total)
}
