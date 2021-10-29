package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/2beens/serjtubincom/internal/file_box"
	"github.com/gorilla/mux"
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
	if r.Method == "OPTIONS" {
		w.Header().Add("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)

	idParam := vars["id"]
	if idParam == "" {
		http.Error(w, "error, file ID empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "error, file ID invalid", http.StatusBadRequest)
		return
	}

	folderIdParam := vars["folderId"]
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.Atoi(folderIdParam)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	fileInfo, err := handler.api.Get(id, folderId)
	log.Debugf("reading from file: %s", fileInfo.Path)

	fileContent, err := os.ReadFile(fileInfo.Path)
	if err != nil {
		log.Errorf("read file [%s]: %s", fileInfo.Path, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(w, bytes.NewReader(fileContent)); err != nil {
		log.Errorf("copy file content for [%s]: %s", fileInfo.Path, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (handler *FileHandler) handleGetRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Add("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	root, err := handler.api.GetFolder(0)
	if err != nil {
		http.Error(w, "internal error", http.StatusBadRequest)
		return
	}

	rootInfo := file_box.NewFolderInfo(root)
	rootInfoJson, err := json.Marshal(rootInfo)
	if err != nil {
		log.Errorf("marshal root folder error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", []byte(rootInfoJson))
}

// handleSave - save file or create a directory
func (handler *FileHandler) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Add("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)

	folderIdParam := vars["folderId"]
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.Atoi(folderIdParam)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	log.Printf("new file upload incoming for folder [%d]", folderId)

	// Maximum upload of 10 MB files
	r.ParseMultipartForm(10 << 20)

	// Get handler for filename, size and headers
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		log.Errorf("get file: %s", err)
		http.Error(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	log.Printf("will try to save file: %s", fileHeader.Filename)

	defer file.Close()
	log.Printf("Uploaded File: %+v\n", fileHeader.Filename)
	log.Printf("File Size: %+v\n", fileHeader.Size)
	log.Printf("MIME Header: %+v\n", fileHeader.Header)
	log.Printf("Content-Type: %+v\n", fileHeader.Header["Content-Type"])

	newFileId, err := handler.api.Save(fileHeader.Filename, folderId, file)
	if err != nil {
		log.Errorf("save new file: %s", err)
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}

	log.Tracef("new file added %d: [%s] added", newFileId, fileHeader.Filename)

	WriteResponse(w, "", fmt.Sprintf("added:%d", newFileId))
}

// handleGetFilesList - return tree structure of a given directory/path
func (handler *FileHandler) handleGetFilesList(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.Header().Add("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)

	folderIdParam := vars["folderId"]
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.Atoi(folderIdParam)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	filesListRaw, err := handler.api.ListFiles(folderId)
	if err != nil {
		http.Error(w, "internal error <sad face>", http.StatusInternalServerError)
		return
	}

	if len(filesListRaw) == 0 {
		WriteResponseBytes(w, "application/json", []byte("[]"))
		return
	}

	var filesList []file_box.FileInfo
	for _, f := range filesListRaw {
		filesList = append(filesList, file_box.FileInfo{
			Id:   f.Id,
			Name: f.Name,
		})
	}

	filesListJson, err := json.Marshal(filesList)
	if err != nil {
		log.Errorf("marshal files list error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", []byte(filesListJson))
}
