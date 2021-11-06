package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/file_box"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type FileHandler struct {
	api          file_box.Api
	loginChecker *auth.LoginChecker
}

func NewFileHandler(api file_box.Api, loginChecker *auth.LoginChecker) *FileHandler {
	return &FileHandler{
		api:          api,
		loginChecker: loginChecker,
	}
}

// handleGet - get file content
func (handler *FileHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
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
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		http.Error(w, "error, file ID invalid", http.StatusBadRequest)
		return
	}

	folderIdParam := vars["folderId"]
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.ParseInt(folderIdParam, 10, 64)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	fileInfo, err := handler.api.Get(id, folderId)
	if err != nil {
		log.Errorf("read file [%d]: %s", id, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	log.Debugf("reading from file: %s", fileInfo.Path)

	if fileInfo.IsPrivate {
		isLogged, err := handler.isLogged(r)
		if err != nil {
			log.Tracef("[file handler] [private file] %s => %s", r.URL.Path, err)
			http.Error(w, "no can do", http.StatusUnauthorized)
			return
		}
		if !isLogged {
			log.Tracef("[invalid token] [private file] unauthorized => %s", r.URL.Path)
			http.Error(w, "no can do", http.StatusUnauthorized)
			return
		}
	}

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

func (handler *FileHandler) handleUpdateFileInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)

	idParam := vars["id"]
	if idParam == "" {
		http.Error(w, "error, file ID empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		http.Error(w, "error, file ID invalid", http.StatusBadRequest)
		return
	}

	folderIdParam := vars["folderId"]
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.ParseInt(folderIdParam, 10, 64)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	newName := r.Form.Get("name")
	isPrivateStr := r.Form.Get("is_private")
	if isPrivateStr == "" {
		http.Error(w, "error, 'is private' empty", http.StatusBadRequest)
		return
	}
	isPrivate := isPrivateStr == "true"

	if err := handler.api.UpdateFileInfo(id, folderId, newName, isPrivate); err != nil {
		log.Errorf("update file info [%d]: %s", id, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("updated:%d", id)))
}

func (handler *FileHandler) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "DELETE, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)

	folderIdParam := vars["folderId"]
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.ParseInt(folderIdParam, 10, 64)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	log.Debugf("--> will try to delete folder [%d]", folderId)

	if err := handler.api.DeleteFolder(folderId); err != nil {
		log.Errorf("delete folder [%d]: %s", folderId, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("deleted:%d", folderId)))
}

// TODO: find out how to set app permissions only for one specific folder and its children
func (handler *FileHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "DELETE, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)

	idParam := vars["id"]
	if idParam == "" {
		http.Error(w, "error, file ID empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		http.Error(w, "error, file ID invalid", http.StatusBadRequest)
		return
	}

	folderIdParam := vars["folderId"]
	if folderIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	folderId, err := strconv.ParseInt(folderIdParam, 10, 64)
	if err != nil {
		http.Error(w, "error, folder ID invalid", http.StatusBadRequest)
		return
	}

	log.Debugf("--> will try to delete file [%d] from folder [%d]", id, folderId)

	fileInfo, err := handler.api.Get(id, folderId)
	if err != nil {
		log.Errorf("delete file [%d]: %s", id, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	log.Debugf("will delete file: %s", fileInfo.Path)

	if err := handler.api.Delete(id, folderId); err != nil {
		log.Errorf("delete file [%d]: %s", id, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("deleted:%d", id)))
}

func (handler *FileHandler) handleGetRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	root, err := handler.api.GetRootFolder()
	if err != nil {
		http.Error(w, "internal error", http.StatusBadRequest)
		return
	}

	rootInfo := file_box.NewFolderInfo(-1, root)
	rootInfoJson, err := json.Marshal(rootInfo)
	if err != nil {
		log.Errorf("marshal root folder error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", []byte(rootInfoJson))
}

func (handler *FileHandler) handleNewFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)

	parentIdParam := vars["parentId"]
	if parentIdParam == "" {
		http.Error(w, "error, folder ID empty", http.StatusBadRequest)
		return
	}
	parentId, err := strconv.ParseInt(parentIdParam, 10, 64)
	if err != nil {
		http.Error(w, "error, parent folder ID invalid", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Errorf("create child folder failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	name := r.Form.Get("name")
	if name == "" {
		http.Error(w, "error, folder name empty", http.StatusBadRequest)
		return
	}

	log.Printf("creating child folder [%s] for folder [%d]", name, parentId)

	if f, err := handler.api.NewFolder(parentId, name); err != nil {
		log.Errorf("create child folder for %d: %s", parentId, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	} else {
		log.Printf("child folder [%d][%s] for folder [%d] created", f.Id, f.Name, parentId)
		WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("created:%d", f.Id)))
	}
}

// handleSave - save file or create a directory
func (handler *FileHandler) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
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
	folderId, err := strconv.ParseInt(folderIdParam, 10, 64)
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

	fileType := "unknown"
	if t, ok := fileHeader.Header["Content-Type"]; ok {
		if len(t) > 0 {
			fileType = t[0]
		}
	}

	newFileId, err := handler.api.Save(
		fileHeader.Filename,
		folderId,
		fileHeader.Size,
		fileType,
		file,
	)
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
	if r.Method == http.MethodOptions {
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
	folderId, err := strconv.ParseInt(folderIdParam, 10, 64)
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

func (handler *FileHandler) isLogged(r *http.Request) (bool, error) {
	authToken := r.Header.Get("X-SERJ-TOKEN")
	if authToken == "" {
		return false, fmt.Errorf("[missing token] unauthorized => %s", r.URL.Path)
	}

	isLogged, err := handler.loginChecker.IsLogged(authToken)
	if err != nil {
		return false, fmt.Errorf("[failed login check] => %s: %s", r.URL.Path, err)
	}

	return isLogged, nil
}

func (handler *FileHandler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				return
			}

			isLogged, err := handler.isLogged(r)
			if err != nil {
				log.Tracef("[file handler] %s => %s", r.URL.Path, err)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}
			if !isLogged {
				log.Tracef("[invalid token] [file handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
