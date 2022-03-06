package file_box

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type FileHandler struct {
	api          *DiskApi
	loginChecker auth.Checker
}

func NewFileHandler(api *DiskApi, loginChecker auth.Checker) *FileHandler {
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
			http.NotFound(w, r)
			return
		}
		if !isLogged {
			log.Tracef("[invalid token] [private file] unauthorized => %s", r.URL.Path)
			http.NotFound(w, r)
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

func (handler *FileHandler) handleUpdateInfo(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseForm(); err != nil {
		log.Errorf("update file info failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	newName := r.Form.Get("name")
	isPrivateStr := r.Form.Get("is_private")
	if isPrivateStr == "" {
		http.Error(w, "error, 'is private' empty", http.StatusBadRequest)
		return
	}
	isPrivate := isPrivateStr == "true"

	if err := handler.api.UpdateInfo(id, folderId, newName, isPrivate); err != nil {
		log.Errorf("update file info [%d]: %s", id, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	internal.WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("updated:%d", id)))
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

	internal.WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("deleted:%d", folderId)))
}

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

	internal.WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("deleted:%d", id)))
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

	rootInfo := NewFolderInfo(-1, root)
	rootInfoJson, err := json.Marshal(rootInfo)
	if err != nil {
		log.Errorf("marshal root folder error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	internal.WriteResponseBytes(w, "application/json", rootInfoJson)
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
		internal.WriteResponseBytes(w, "application/json", []byte(fmt.Sprintf("created:%d", f.Id)))
	}
}

// handleUpload - save file or create a directory
func (handler *FileHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
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

	const maxFileSize = 1024 * 1024 * 999 // 999 MB
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		log.Errorf("get file, parse multipart form: %s", err)
		http.Error(w, "internal error or file too big", http.StatusInternalServerError)
		return
	}

	var addedFileIds []string
	files := r.MultipartForm.File["files"]
	for _, fileHeader := range files {
		log.Printf("trying to save file: %s", fileHeader.Filename)
		file, err := fileHeader.Open()
		if err != nil {
			log.Errorf("upload file: %s", err)
			http.Error(w, "failed to upload file [cannot open file]", http.StatusInternalServerError)
			return
		}

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
			log.Errorf("upload new file: %s", err)
			http.Error(w, "failed to upload file", http.StatusInternalServerError)
			file.Close()
			return
		}

		addedFileIds = append(addedFileIds, fmt.Sprintf("%d", newFileId))

		if err := file.Close(); err != nil {
			log.Errorf("failed to close file properly [%s]: %s", fileHeader.Filename, err)
		}

		log.Tracef("new file added %d: [%s] added", newFileId, fileHeader.Filename)
	}

	internal.WriteResponse(w, "", fmt.Sprintf("added:%s", strings.Join(addedFileIds, ",")))
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
				http.NotFound(w, r) // deliberately return not found
				return
			}
			if !isLogged {
				log.Tracef("[invalid token] [file handler] unauthorized => %s", r.URL.Path)
				http.NotFound(w, r) // deliberately return not found
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
