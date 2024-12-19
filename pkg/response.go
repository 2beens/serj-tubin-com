package pkg

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const (
	ContentTypeJSON = "application/json"
	ContentTypeText = "text/plain"
)

var ContentType = struct {
	JSON string
	Text string
}{
	JSON: ContentTypeJSON,
	Text: ContentTypeText,
}

func WriteResponse(w http.ResponseWriter, contentType, message string, statusCode int) {
	WriteResponseBytes(w, contentType, []byte(message), statusCode)
}

func WriteResponseOK(w http.ResponseWriter, contentType, message string) {
	WriteResponse(w, contentType, message, http.StatusOK)
}

func WriteJSONResponseOK(w http.ResponseWriter, json string) {
	WriteResponseOK(w, ContentType.JSON, json)
}

func WriteTextResponseOK(w http.ResponseWriter, text string) {
	WriteResponseOK(w, ContentType.Text, text)
}

func WriteResponseBytes(w http.ResponseWriter, contentType string, message []byte, statusCode int) {
	if contentType != "" {
		w.Header().Add("Content-Type", contentType)
	}

	w.WriteHeader(statusCode)

	if _, err := w.Write(message); err != nil {
		log.Errorf("failed to write response [%s]: %s", message, err)
	}
}

func WriteResponseBytesOK(w http.ResponseWriter, contentType string, message []byte) {
	WriteResponseBytes(w, contentType, message, http.StatusOK)
}

func SendJsonResponse(w http.ResponseWriter, statusCode int, value any) {
	valueJson, err := json.Marshal(value)
	if err != nil {
		log.Errorf("failed to marshal value to json: %v", err)
		http.Error(w, "failed to marshal value to json", http.StatusInternalServerError)
		return
	}
	WriteResponseBytes(w, ContentType.JSON, valueJson, statusCode)
}
