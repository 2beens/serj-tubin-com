package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *IntegrationTestSuite) TestFileBox() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t := s.T()
	token := s.doLogin(ctx)

	fileServiceEndpoint := "http://localhost:9001"

	// 1. Create a new folder
	// POST /f/{parentId}/new
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/f/0/new", fileServiceEndpoint), nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", token)
	req.Header.Set("Origin", "http://localhost")
	req.PostForm = map[string][]string{"name": {"test-folder"}}

	resp, err := s.httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// response format: created:{id}
	respStr := string(respBody)
	require.Contains(t, respStr, "created:")
	folderIdStr := strings.TrimPrefix(respStr, "created:")
	fmt.Printf("created folder id: %s\n", folderIdStr)

	// 2. Upload a file to the new folder
	// POST /f/upload/{folderId}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("files", "test-file.txt")
	require.NoError(t, err)
	_, err = io.Copy(fw, strings.NewReader("test content"))
	require.NoError(t, err)
	w.Close()

	req, err = http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/f/upload/%s", fileServiceEndpoint, folderIdStr), &b)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", token)
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err = s.httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	respBody, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	// response format: added:{id}
	respStr = string(respBody)
	require.Contains(t, respStr, "added:")
	fileIdStr := strings.TrimPrefix(respStr, "added:")
	fmt.Printf("uploaded file id: %s\n", fileIdStr)

	// 3. Get the file content
	// GET /f/link/{id}
	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/link/%s", fileServiceEndpoint, fileIdStr), nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", token)
	req.Header.Set("Origin", "http://localhost")

	resp, err = s.httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	respBody, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "test content", string(respBody))

	// 4. Delete the file
	// POST /f/del
	req, err = http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/f/del", fileServiceEndpoint), nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", token)
	req.Header.Set("Origin", "http://localhost")
	req.PostForm = map[string][]string{"ids": {fileIdStr}}

	resp, err = s.httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	respBody, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "deleted:1", string(respBody))

	// 5. Verify file is gone
	req, err = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/link/%s", fileServiceEndpoint, fileIdStr), nil)
	require.NoError(t, err)
	req.Header.Set("X-SERJ-TOKEN", token)
	req.Header.Set("Origin", "http://localhost")
	resp, err = s.httpClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode) // currently returns 500 on file not found in handleGet, or 404 if handled
	resp.Body.Close()
}
