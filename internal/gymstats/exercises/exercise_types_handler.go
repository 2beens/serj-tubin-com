package exercises

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/file_box"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var MuscleGroup = struct {
	Biceps    string
	Triceps   string
	Back      string
	Legs      string
	Chest     string
	Shoulders string
	Other     string
}{
	Biceps:    "biceps",
	Triceps:   "triceps",
	Back:      "back",
	Legs:      "legs",
	Chest:     "chest",
	Shoulders: "shoulders",
	Other:     "other",
}

var MuscleGroups = []string{
	MuscleGroup.Biceps,
	MuscleGroup.Triceps,
	MuscleGroup.Back,
	MuscleGroup.Legs,
	MuscleGroup.Chest,
	MuscleGroup.Shoulders,
	MuscleGroup.Other,
}

//go:generate mockgen -source=$GOFILE -destination=exercise_types_mocks_test.go -package=exercises_test

type exerciseTypesRepo interface {
	GetExerciseType(ctx context.Context, exerciseTypeID string) (_ ExerciseType, err error)
	GetExerciseTypes(ctx context.Context, params GetExerciseTypesParams) (_ []ExerciseType, err error)
	AddExerciseType(ctx context.Context, exerciseType ExerciseType) (err error)
	AddExerciseTypeImage(ctx context.Context, exerciseImage ExerciseImage) (err error)
	UpdateExerciseType(ctx context.Context, exerciseType ExerciseType) (err error)
	DeleteExerciseType(ctx context.Context, exerciseTypeID string) (err error)
	DeleteExerciseTypeImage(ctx context.Context, exerciseImageID int64) (err error)
}

type TypesHandler struct {
	diskApi *file_box.DiskApi // used for storing/getting exercise type images
	repo    exerciseTypesRepo
}

func NewTypesHandler(
	diskApi *file_box.DiskApi,
	repo exerciseTypesRepo,
) *TypesHandler {
	return &TypesHandler{
		diskApi: diskApi,
		repo:    repo,
	}
}

func (handler *TypesHandler) HandleAdd(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercise_types.new")
	defer span.End()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var exerciseType ExerciseType
	if err := json.NewDecoder(r.Body).Decode(&exerciseType); err != nil {
		log.Errorf("new exercise type, unmarshal json params: %s", err)
		http.Error(w, "add exercise type failed", http.StatusBadRequest)
		return
	}

	if exerciseType.ID == "" || exerciseType.MuscleGroup == "" || exerciseType.Name == "" {
		http.Error(w, "error, exercise id, muscle group, and name are required", http.StatusBadRequest)
		return
	}

	exerciseType.MuscleGroup = strings.ToLower(exerciseType.MuscleGroup)
	if slices.Contains(MuscleGroups, exerciseType.MuscleGroup) == false {
		http.Error(w, "error, invalid muscle group", http.StatusBadRequest)
		return
	}

	if exerciseType.CreatedAt.IsZero() {
		exerciseType.CreatedAt = time.Now()
	}

	if err := handler.repo.AddExerciseType(ctx, exerciseType); err != nil {
		log.Errorf("add exercise type: %s", err)
		http.Error(w, "add exercise type failed", http.StatusInternalServerError)
		return
	}

	log.Debugf("new exercise type added: %+v", exerciseType)
	w.WriteHeader(http.StatusCreated)
}

func (handler *TypesHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercise_types.get")
	defer span.End()

	exerciseTypes, err := handler.repo.GetExerciseTypes(ctx, GetExerciseTypesParams{
		MuscleGroup: r.URL.Query().Get("muscleGroup"),
		ExerciseId:  r.URL.Query().Get("id"),
	})
	if err != nil {
		log.Errorf("get exercise types: %s", err)
		http.Error(w, "get exercise types failed", http.StatusInternalServerError)
		return
	}

	exTypesJson, err := json.Marshal(exerciseTypes)
	if err != nil {
		log.Errorf("marshal exercise types: %s", err)
		http.Error(w, "get exercise types failed", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, exTypesJson, http.StatusOK)
}

func (handler *TypesHandler) HandleGetImage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercise_types.get_image")
	defer span.End()

	vars := mux.Vars(r)
	idParam := vars["id"]
	if idParam == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		http.Error(w, "error, file ID invalid", http.StatusBadRequest)
		return
	}

	log.Debugf("get image, id: %d", id)

	imageFile, _, err := handler.diskApi.Get(ctx, id)
	if err != nil {
		log.Errorf("get image: %s", err)
		http.Error(w, "get image failed", http.StatusInternalServerError)
		return
	}

	log.Debugf("image %s found, serving...", imageFile.Path)

	http.ServeFile(w, r, imageFile.Path)
}

func (handler *TypesHandler) HandleDeleteImage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercise_types.delete_image")
	defer span.End()

	vars := mux.Vars(r)
	idParam := vars["id"]
	if idParam == "" {
		http.Error(w, "error, image ID empty", http.StatusBadRequest)
		return
	}
	imageId, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		http.Error(w, "error, image ID invalid", http.StatusBadRequest)
		return
	}

	log.Debugf("delete image %d", imageId)

	if err := handler.repo.DeleteExerciseTypeImage(ctx, imageId); err != nil {
		if errors.Is(err, ErrExerciseTypeNotFound) {
			http.Error(w, "delete image failed - not found", http.StatusNotFound)
			return
		}
		log.Errorf("delete image: %s", err)
		http.Error(w, "delete image failed", http.StatusInternalServerError)
		return
	}

	if err := handler.diskApi.Delete(ctx, imageId); err != nil {
		log.Errorf("delete image: %s", err)
		http.Error(w, "delete image failed", http.StatusInternalServerError)
		return
	}

	log.Debugf("image %d deleted", imageId)
	w.WriteHeader(http.StatusNoContent)
}

func (handler *TypesHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercise_types.update")
	defer span.End()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var exerciseType ExerciseType
	if err := json.NewDecoder(r.Body).Decode(&exerciseType); err != nil {
		log.Errorf("update exercise type, unmarshal json params: %s", err)
		http.Error(w, "update exercise type failed", http.StatusBadRequest)
		return
	}

	if exerciseType.ID == "" || exerciseType.MuscleGroup == "" || exerciseType.Name == "" {
		http.Error(w, "error, exercise id, muscle group, and name are required", http.StatusBadRequest)
		return
	}

	if err := handler.repo.UpdateExerciseType(ctx, exerciseType); err != nil {
		log.Errorf("update exercise type: %s", err)
		http.Error(w, "update exercise type failed", http.StatusInternalServerError)
		return
	}

	log.Debugf("exercise type updated: %+v", exerciseType)
	w.WriteHeader(http.StatusNoContent)
}

func (handler *TypesHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercise_types.delete")
	defer span.End()

	vars := mux.Vars(r)
	id := vars["id"]
	if id == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}

	if err := handler.repo.DeleteExerciseType(ctx, id); err != nil {
		log.Errorf("delete exercise type: %s", err)
		http.Error(w, "delete exercise type failed", http.StatusInternalServerError)
		return
	}

	log.Debugf("exercise type deleted: %s", id)
	w.WriteHeader(http.StatusNoContent)
}

type SavedImageResponse struct {
	ID int64 `json:"id"`
}

func (handler *TypesHandler) HandleUploadImage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercise_types.upload_image")
	defer span.End()

	vars := mux.Vars(r)
	exerciseTypeID := vars["id"]
	if exerciseTypeID == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}

	_, err := handler.repo.GetExerciseType(ctx, exerciseTypeID)
	if err != nil {
		log.Errorf("upload image, get exercise type: %s", err)
		http.Error(w, "upload image failed, exercise type not found", http.StatusNotFound)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		log.Errorf("upload image, get file from form: %s", err)
		http.Error(w, "upload image failed", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Errorf("upload image, close file: %s", err)
		}
	}()

	log.Debugf(
		"upload image, filename: %s, size: %d, content-type: %s",
		header.Filename, header.Size, header.Header["Content-Type"],
	)

	rootFolder, err := handler.diskApi.GetRootFolder()
	if err != nil {
		log.Errorf("upload image, get root folder: %s", err)
		http.Error(w, "upload image failed", http.StatusInternalServerError)
		return
	}

	var imagesFolder *file_box.Folder
	for _, f := range rootFolder.Subfolders {
		if f.Name == "images" {
			imagesFolder = f
			break
		}
	}

	fileType := "unknown"
	if t, ok := header.Header["Content-Type"]; ok {
		if len(t) > 0 {
			fileType = t[0]
		}
	}

	uploadedFileId, err := handler.diskApi.Save(
		ctx, file_box.SaveFileParams{
			Filename:  header.Filename,
			FolderId:  imagesFolder.Id,
			Size:      header.Size,
			FileType:  fileType,
			File:      file,
			IsPrivate: false,
		},
	)
	if err != nil {
		log.Errorf("upload image, save file: %s", err)
		http.Error(w, "upload image failed", http.StatusInternalServerError)
		return
	}

	// store the image metadata to the database
	exerciseImage := ExerciseImage{
		ID:         uploadedFileId,
		ExerciseID: exerciseTypeID,
		CreatedAt:  time.Now(),
	}
	if err := handler.repo.AddExerciseTypeImage(ctx, exerciseImage); err != nil {
		log.Errorf("upload image, save image metadata: %s", err)
		http.Error(w, "upload image failed", http.StatusInternalServerError)
		return
	}

	savedImageJson, err := json.Marshal(SavedImageResponse{ID: uploadedFileId})
	if err != nil {
		log.Errorf("upload image, marshal saved image: %s", err)
		http.Error(w, "upload image failed", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, savedImageJson, http.StatusCreated)
}
