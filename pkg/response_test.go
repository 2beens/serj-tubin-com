package pkg

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestHttpResponseWriter struct {
	HeaderMap  http.Header
	Body       []byte
	StatusCode int
}

func (w *TestHttpResponseWriter) Header() http.Header {
	return w.HeaderMap
}

func (w *TestHttpResponseWriter) Write(bytes []byte) (int, error) {
	w.Body = bytes
	return len(bytes), nil
}

func (w *TestHttpResponseWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
}

func TestWriteResponseBytes(t *testing.T) {
	w := &TestHttpResponseWriter{
		HeaderMap: make(http.Header),
	}

	testJson := `{"key":"val"}`
	WriteResponseBytes(w, ContentType.JSON, []byte(testJson), http.StatusOK)

	assert.Equal(t, http.StatusOK, w.StatusCode)
	assert.Equal(t, ContentType.JSON, w.HeaderMap.Get("Content-Type"))
	assert.Equal(t, testJson, string(w.Body))
}

func TestWriteResponseBytesOK(t *testing.T) {
	w := &TestHttpResponseWriter{
		HeaderMap: make(http.Header),
	}

	testJson := `{"key":"val"}`
	WriteResponseBytesOK(w, ContentType.JSON, []byte(testJson))

	assert.Equal(t, http.StatusOK, w.StatusCode)
	assert.Equal(t, ContentType.JSON, w.HeaderMap.Get("Content-Type"))
	assert.Equal(t, testJson, string(w.Body))
}

func TestWriteResponse(t *testing.T) {
	w := &TestHttpResponseWriter{
		HeaderMap: make(http.Header),
	}

	testJson := `{"key":"val"}`
	WriteResponse(w, ContentType.JSON, testJson, http.StatusOK)

	assert.Equal(t, http.StatusOK, w.StatusCode)
	assert.Equal(t, ContentType.JSON, w.HeaderMap.Get("Content-Type"))
	assert.Equal(t, testJson, string(w.Body))
}

func TestWriteTextResponseOK(t *testing.T) {
	w := &TestHttpResponseWriter{
		HeaderMap: make(http.Header),
	}

	testText := `test text`
	WriteTextResponseOK(w, testText)

	assert.Equal(t, http.StatusOK, w.StatusCode)
	assert.Equal(t, ContentType.Text, w.HeaderMap.Get("Content-Type"))
	assert.Equal(t, testText, string(w.Body))
}

func TestWriteJSONResponseOK(t *testing.T) {
	w := &TestHttpResponseWriter{
		HeaderMap: make(http.Header),
	}

	testJson := `{"key":"val"}`
	WriteJSONResponseOK(w, testJson)

	assert.Equal(t, http.StatusOK, w.StatusCode)
	assert.Equal(t, ContentType.JSON, w.HeaderMap.Get("Content-Type"))
	assert.Equal(t, testJson, string(w.Body))
}
