package internal

import (
	"fmt"
	"net/http"

	"github.com/2beens/serjtubincom/internal/file_box"
	log "github.com/sirupsen/logrus"
)

type FileHandler struct {
	api file_box.Api
}

func NewFileHandler(api file_box.Api) *FileHandler {
	return &FileHandler{
		api: api,
	}
}

// handleGet - get file content
func (handler *FileHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("add new note failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	folderId := r.Form.Get("folderId")
	if folderId == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}

	// TODO:
}

// handleSave - save file or create a directory
func (handler *FileHandler) handleSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("add new note failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	folderId := r.Form.Get("folderId")
	if folderId == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}

	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	// Get handler for filename, size and headers
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		log.Printf("get file: %s", err)
		http.Error(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	log.Printf("will try to save file: %s", fileHeader.Filename)

	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", fileHeader.Filename)
	fmt.Printf("File Size: %+v\n", fileHeader.Size)
	fmt.Printf("MIME Header: %+v\n", fileHeader.Header)

	// TODO
	dirId := 1000

	if err := handler.api.Save(fileHeader.Filename, dirId, file); err != nil {
		// TODO;
		fmt.Print(err)
	}

	fmt.Fprintf(w, "Successfully Uploaded File\n")
}

// handleGetFilesList - return tree structure of a given directory/path
func (handler *FileHandler) handleGetFilesList(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("add new note failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	folderId := r.Form.Get("folderId")
	if folderId == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}

	// TODO:
}
