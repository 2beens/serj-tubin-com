package file_box

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileHandler(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := NewDiskApi(tempRootDir)
	require.NoError(t, err)

	loginChecker := auth.NewLoginTestChecker()
	fileHandler := NewFileHandler(api, loginChecker)
	assert.NotNil(t, fileHandler)
}

func TestNewFileHandler_handleGet(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := NewDiskApi(tempRootDir)
	require.NoError(t, err)

	var addedFiles []int64
	parentId := int64(0) // root = id 0
	filesLen := 10
	for i := 1; i <= filesLen; i++ {
		randomContent := strings.NewReader(fmt.Sprintf("random test content %d", i))
		fileName := fmt.Sprintf("file_%d", i)
		fileId, err := api.Save(
			fileName,
			parentId,
			randomContent.Size(),
			"rand-binary",
			randomContent,
		)
		require.NoError(t, err)
		assert.True(t, fileId > 0)

		// make the first 5 files not private
		if i <= 5 {
			require.NoError(t, api.UpdateInfo(fileId, parentId, fileName, false))
		}

		addedFiles = append(addedFiles, fileId)
	}
	assert.Len(t, api.root.Files, filesLen)
	require.Len(t, addedFiles, filesLen)

	loginChecker := auth.NewLoginTestChecker()
	fileHandler := NewFileHandler(api, loginChecker)

	r := mux.NewRouter()
	r.HandleFunc("/link/{folderId}/c/{id}", fileHandler.handleGet).Methods("GET", "OPTIONS")

	req, err := http.NewRequest("GET", fmt.Sprintf("/link/0/c/%d", addedFiles[4]), nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, "random test content 5", rr.Body.String())

	req, err = http.NewRequest("GET", fmt.Sprintf("/link/0/c/%d", addedFiles[0]), nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, "random test content 1", rr.Body.String())

	// private file - should not return anything
	req, err = http.NewRequest("GET", fmt.Sprintf("/link/0/c/%d", addedFiles[8]), nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "404 page not found\n", rr.Body.String())
}
