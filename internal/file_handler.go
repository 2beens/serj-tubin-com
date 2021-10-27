package internal

import (
	"fmt"
	"net/http"
	"strconv"

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

	idParam := r.Form.Get("id")
	if idParam == "" {
		http.Error(w, "error, file ID empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "error, file ID invalid", http.StatusBadRequest)
		return
	}

	folderIdParam := r.Form.Get("folderId")
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.Atoi(folderIdParam)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	file, err := handler.api.Get(id, folderId)
	log.Debugf("reading from file: %s", file.Path)

	// TODO:

}

// handleSave - save file or create a directory
func (handler *FileHandler) handleSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("add new note failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	folderIdParam := r.Form.Get("folderId")
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.Atoi(folderIdParam)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
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

	newFileId, err := handler.api.Save(fileHeader.Filename, folderId, file)
	if err != nil {
		log.Printf("save new file: %s", err)
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	log.Tracef("new file added %d: [%s] added", newFileId, fileHeader.Filename)

	WriteResponse(w, "", fmt.Sprintf("added:%d", newFileId))
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
