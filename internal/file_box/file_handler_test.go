package file_box

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/2beens/serjtubincom/internal/auth"
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

func TestFileHandler_handleGet(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := NewDiskApi(tempRootDir)
	require.NoError(t, err)
	require.NotNil(t, api)

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
			require.NoError(t, api.UpdateInfo(fileId, fileName, false))
		}

		addedFiles = append(addedFiles, fileId)
	}
	assert.Len(t, api.root.Files, filesLen)
	require.Len(t, addedFiles, filesLen)

	loginChecker := auth.NewLoginTestChecker()
	fileHandler := NewFileHandler(api, loginChecker)

	r := RouterSetup(fileHandler)

	req, err := http.NewRequest("GET", fmt.Sprintf("/link/%d", addedFiles[4]), nil)
	require.NoError(t, err)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, "random test content 5", rr.Body.String())

	req, err = http.NewRequest("GET", fmt.Sprintf("/link/%d", addedFiles[0]), nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, "random test content 1", rr.Body.String())

	// private file - should not return anything
	req, err = http.NewRequest("GET", fmt.Sprintf("/link/%d", addedFiles[8]), nil)
	require.NoError(t, err)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "404 page not found\n", rr.Body.String())

	// private file, but logged in - should return the file
	loginChecker.LoggedSessions["test-token"] = true
	req, err = http.NewRequest("GET", fmt.Sprintf("/link/%d", addedFiles[8]), nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "test-token")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, "random test content 9", rr.Body.String())

	// private file, but logged out - should not return the file
	loginChecker.LoggedSessions["test-token"] = false
	req, err = http.NewRequest("GET", fmt.Sprintf("/link/%d", addedFiles[8]), nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "test-token")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "404 page not found\n", rr.Body.String())
}

func TestFileHandler_handleDeleteFile(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := NewDiskApi(tempRootDir)
	require.NoError(t, err)
	require.NotNil(t, api)

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
			require.NoError(t, api.UpdateInfo(fileId, fileName, false))
		}

		addedFiles = append(addedFiles, fileId)
	}
	assert.Len(t, api.root.Files, filesLen)
	require.Len(t, addedFiles, filesLen)

	loginChecker := auth.NewLoginTestChecker()
	loginChecker.LoggedSessions["test-token"] = true
	fileHandler := NewFileHandler(api, loginChecker)

	r := RouterSetup(fileHandler)

	// before delete, file there?
	file1, parent, err := api.Get(addedFiles[0])
	require.NoError(t, err)
	assert.NotNil(t, file1)
	assert.Equal(t, parentId, parent.Id)

	req, err := http.NewRequest("POST", "/f/del", nil)
	require.NoError(t, err)
	req.PostForm = url.Values{}
	req.PostForm.Add("ids", fmt.Sprintf("%d,%d", addedFiles[0], addedFiles[2]))
	req.Header.Set("X-SERJ-TOKEN", "test-token")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, fmt.Sprintf("deleted:%d", 2), rr.Body.String())

	assert.Len(t, api.root.Files, filesLen-2)

	file, parent, err := api.Get(addedFiles[0])
	assert.ErrorIs(t, err, ErrFileNotFound)
	assert.Nil(t, file)
	assert.Nil(t, parent)
	file, parent, err = api.Get(addedFiles[2])
	assert.ErrorIs(t, err, ErrFileNotFound)
	assert.Nil(t, file)
	assert.Nil(t, parent)
}

func TestFileHandler_handleUpdateInfo(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := NewDiskApi(tempRootDir)
	require.NoError(t, err)
	require.NotNil(t, api)

	parentId := int64(0) // root = id 0
	fileContentString := "random test file content"
	fileContent := strings.NewReader(fileContentString)
	fileName := "test-name"
	fileId, err := api.Save(
		fileName,
		parentId,
		fileContent.Size(),
		"rand-binary",
		fileContent,
	)
	require.NoError(t, err)
	assert.True(t, fileId > 0)
	assert.Len(t, api.root.Files, 1)

	loginChecker := auth.NewLoginTestChecker()
	loginChecker.LoggedSessions["test-token"] = true
	fileHandler := NewFileHandler(api, loginChecker)

	r := RouterSetup(fileHandler)

	req, err := http.NewRequest("POST", fmt.Sprintf("/f/update/%d", fileId), nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "test-token")
	req.PostForm = url.Values{}
	req.PostForm.Add("is_private", "false")
	req.PostForm.Add("name", "safari")

	// before
	file := api.root.Files[fileId]
	assert.Equal(t, fileName, file.Name)
	assert.True(t, file.IsPrivate)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, fmt.Sprintf("updated:%d", fileId), rr.Body.String())

	// after
	assert.Equal(t, "safari", file.Name)
	assert.False(t, file.IsPrivate)
	fileContentRetrieved, err := os.ReadFile(file.Path)
	require.NoError(t, err)
	assert.Equal(t, fileContentString, string(fileContentRetrieved))
}

func TestFileHandler_handleGetRoot(t *testing.T) {
	tempRootDir := t.TempDir()
	api, err := NewDiskApi(tempRootDir)
	require.NoError(t, err)
	require.NotNil(t, api)

	rootId := int64(0)
	fileContent := strings.NewReader("random test file content")
	fileName := "test-name"
	fileId, err := api.Save(
		fileName,
		rootId,
		fileContent.Size(),
		"rand-binary",
		fileContent,
	)
	require.NoError(t, err)
	assert.True(t, fileId > 0)
	assert.Len(t, api.root.Files, 1)

	loginChecker := auth.NewLoginTestChecker()
	loginChecker.LoggedSessions["test-token"] = true
	fileHandler := NewFileHandler(api, loginChecker)

	r := RouterSetup(fileHandler)

	req, err := http.NewRequest("GET", "/f/root", nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "test-token")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	rootJson := rr.Body.String()
	require.NotEmpty(t, rootJson)

	var retrievedRoot FileInfo
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &retrievedRoot))
	require.NotNil(t, retrievedRoot)
	require.Len(t, retrievedRoot.Children, 1)
	assert.False(t, retrievedRoot.IsFile)
	assert.Equal(t, retrievedRoot.Children[0].Name, fileName)
	assert.True(t, retrievedRoot.Children[0].IsPrivate)
	assert.True(t, retrievedRoot.Children[0].IsFile)

	// now log out and try - no root should return
	loginChecker.LoggedSessions["test-token"] = false
	req, err = http.NewRequest("GET", "/f/root", nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", "test-token")

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "404 page not found\n", rr.Body.String())

	// now missing token - no root should return
	req, err = http.NewRequest("GET", "/f/root", nil)
	require.NoError(t, err)

	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "404 page not found\n", rr.Body.String())
}

// TODO: add the rest :)
