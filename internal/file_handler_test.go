package internal

import (
	"testing"

	"net/http"
	"net/http/httptest"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/file_box"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileHandler(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := file_box.NewDiskTestApi(tempRootDir)
	require.NoError(t, err)

	loginChecker := auth.NewLoginTestChecker()
	fileHandler := NewFileHandler(api, loginChecker)
	assert.NotNil(t, fileHandler)
}

func TestNewFileHandler_handleGet(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := file_box.NewDiskTestApi(tempRootDir)
	require.NoError(t, err)

	loginChecker := auth.NewLoginTestChecker()
	fileHandler := NewFileHandler(api, loginChecker)

	r := mux.NewRouter()
	r.HandleFunc("/link/{folderId}/c/{id}", fileHandler.handleGet).Methods("GET", "OPTIONS")

	req, err := http.NewRequest("GET", "/link/0/c/100", nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// TODO:
}
