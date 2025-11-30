package file_box

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

const maxUploadedFileSize = 1024 * 1024 * 999 // 999 MB

type deleteRequest struct {
	Ids []int64 `json:"ids"`
}

type updateInfoRequest struct {
	Name      string `json:"name"`
	IsPrivate bool   `json:"is_private"`
}

type newFolderRequest struct {
	Name string `json:"name"`
}

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

func (handler *FileHandler) handleDownloadFolder(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.downloadFolder")
	defer span.End()

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

	log.Debugf("--> will try to download folder [%d]", folderId)

	folder, err := handler.api.GetFolder(ctx, folderId)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	span.SetAttributes(attribute.String("folder.name", folder.Name))

	w.Header().Set("Content-Type", "application/zip")
	if err := pkg.Compress(folder.Path, w); err != nil {
		log.Errorf("compress folder [%s]: %s", folder.Path, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (handler *FileHandler) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.downloadFile")
	defer span.End()

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

	log.Debugf("--> will try to download file [%d]", id)

	file, _, err := handler.api.Get(ctx, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	if err := pkg.Compress(file.Path, w); err != nil {
		log.Errorf("compress file [%s]: %s", file.Path, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

// handleGet - get file content
func (handler *FileHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.get")
	defer span.End()

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

	fileInfo, _, err := handler.api.Get(ctx, id)
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

	file, err := os.Open(fileInfo.Path)
	if err != nil {
		log.Errorf("open file [%s]: %s", fileInfo.Path, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	http.ServeContent(w, r, fileInfo.Name, fileInfo.CreatedAt, file)
}

func (handler *FileHandler) handleUpdateInfo(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.updateInfo")
	defer span.End()

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

	var updateInfoReq updateInfoRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&updateInfoReq); err != nil {
			log.Errorf("update file info failed, decode request error: %s", err)
			http.Error(w, "update file info failed", http.StatusInternalServerError)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("update file info failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}
		isPrivateStr := r.Form.Get("is_private")
		if isPrivateStr == "" {
			http.Error(w, "error, 'is private' empty", http.StatusBadRequest)
			return
		}
		updateInfoReq = updateInfoRequest{
			Name:      r.Form.Get("name"),
			IsPrivate: isPrivateStr == "true",
		}
	}

	if err := handler.api.UpdateInfo(ctx, id, updateInfoReq.Name, updateInfoReq.IsPrivate); err != nil {
		log.Errorf("update file info [%d]: %s", id, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	pkg.WriteTextResponseOK(w, fmt.Sprintf("updated:%d", id))
}

func (handler *FileHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.delete")
	defer span.End()

	var deleteReq deleteRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&deleteReq); err != nil {
			log.Errorf("delete files/folders, decode request error: %s", err)
			http.Error(w, "delete files/folders failed", http.StatusInternalServerError)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("delete files/folders, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}
		idsParam := r.Form.Get("ids")
		if idsParam == "" {
			http.Error(w, "error, IDs parameter empty", http.StatusBadRequest)
			return
		}
		idsRaw := strings.Split(idsParam, ",")
		var ids []int64
		for _, idRaw := range idsRaw {
			id, err := strconv.ParseInt(idRaw, 10, 64)
			if err != nil {
				http.Error(w, fmt.Sprintf("error, file/folder ID [%s] invalid", idRaw), http.StatusBadRequest)
				return
			}
			ids = append(ids, id)
		}
		deleteReq = deleteRequest{
			Ids: ids,
		}
	}

	log.Debugf("--> will try to delete [%d] items", len(deleteReq.Ids))

	deletedCount := 0
	for _, id := range deleteReq.Ids {
		log.Debugf("-> tryint to delete item: %d", id)

		fileInfo, _, err := handler.api.Get(ctx, id)
		if err != nil && !errors.Is(err, ErrFileNotFound) {
			log.Errorf("delete file [%d]: %s", id, err)
			//http.Error(w, "internal error", http.StatusInternalServerError)
			//return
		} else if fileInfo != nil {
			log.Debugf("will delete file: %s", fileInfo.Path)
			if err := handler.api.Delete(ctx, id); err != nil {
				log.Errorf("delete file [%d]: %s", id, err)
				//http.Error(w, "internal error", http.StatusInternalServerError)
				//return
				continue
			}
			deletedCount++
		} else {
			// not a file - try to delete folder instead
			if err := handler.api.DeleteFolder(ctx, id); err != nil {
				log.Errorf("delete folder [%d]: %s", id, err)
				//http.Error(w, "internal error", http.StatusInternalServerError)
				//return
				continue
			}
			deletedCount++
		}
	}

	pkg.WriteTextResponseOK(w, fmt.Sprintf("deleted:%d", deletedCount))
}

func (handler *FileHandler) handleGetRoot(w http.ResponseWriter, r *http.Request) {
	_, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.getRoot")
	defer span.End()

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

	pkg.WriteJSONResponseOK(w, string(rootInfoJson))
}

func (handler *FileHandler) handleNewFolder(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.newFolder")
	defer span.End()

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

	var newFolderReq newFolderRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&newFolderReq); err != nil {
			log.Errorf("create new folder, decode request: %s", err)
			http.Error(w, "create new folder failed", http.StatusInternalServerError)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("create child folder failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}
		newFolderReq = newFolderRequest{
			Name: r.Form.Get("name"),
		}
	}

	if newFolderReq.Name == "" {
		http.Error(w, "error, folder name empty", http.StatusBadRequest)
		return
	}

	log.Debugf("creating child folder [%s] for folder [%d]", newFolderReq.Name, parentId)

	if f, err := handler.api.NewFolder(ctx, parentId, newFolderReq.Name); err != nil {
		log.Errorf("create child folder for %d: %s", parentId, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	} else {
		log.Debugf("child folder [%d][%s] for folder [%d] created", f.Id, f.Name, parentId)
		pkg.WriteResponse(w, pkg.ContentType.Text, fmt.Sprintf("created:%d", f.Id), http.StatusCreated)
	}
}

// handleUpload - save file or create a directory
func (handler *FileHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.upload")
	defer span.End()

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

	log.Tracef("new file upload incoming for folder [%d]", folderId)

	if err := r.ParseMultipartForm(maxUploadedFileSize); err != nil {
		log.Errorf("get file, parse multipart form: %s", err)
		http.Error(w, "internal error or file too big", http.StatusInternalServerError)
		return
	}

	var addedFileIds []string
	files := r.MultipartForm.File["files"]
	for _, fileHeader := range files {
		log.Debugf("trying to save file: %s", fileHeader.Filename)
		file, err := fileHeader.Open()
		if err != nil {
			log.Errorf("upload file: %s", err)
			http.Error(w, "failed to upload file [cannot open file]", http.StatusInternalServerError)
			return
		}

		log.Debugf("File Size: %+v\n", fileHeader.Size)
		log.Debugf("MIME Header: %+v\n", fileHeader.Header)
		log.Debugf("Content-Type: %+v\n", fileHeader.Header["Content-Type"])

		fileType := "unknown"
		if t, ok := fileHeader.Header["Content-Type"]; ok {
			if len(t) > 0 {
				fileType = t[0]
			}
		}

		newFileId, err := handler.api.Save(
			ctx,
			SaveFileParams{
				Filename:  fileHeader.Filename,
				FolderId:  folderId,
				Size:      fileHeader.Size,
				FileType:  fileType,
				File:      file,
				IsPrivate: true,
			},
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

	pkg.WriteTextResponseOK(
		w,
		fmt.Sprintf("added:%s", strings.Join(addedFileIds, ",")),
	)
}

func (handler *FileHandler) isLogged(r *http.Request) (bool, error) {
	authToken := r.Header.Get("X-SERJ-TOKEN")
	if authToken == "" {
		return false, fmt.Errorf("[missing token] unauthorized => %s", r.URL.Path)
	}

	isLogged, err := handler.loginChecker.IsLogged(r.Context(), authToken)
	if err != nil {
		return false, fmt.Errorf("[failed login check] => %s: %s", r.URL.Path, err)
	}

	return isLogged, nil
}

func (handler *FileHandler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Add("Allow", "GET, POST, OPTIONS")
				w.WriteHeader(http.StatusOK)
				return
			}

			ctx, span := tracing.GlobalTracer.Start(r.Context(), "fileHandler.auth")
			defer span.End()

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

			// pass the tracing info from ctx to r.Context()
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}
