package exercises

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"
)

//go:generate mockgen -source=$GOFILE -destination=exercises_mocks_test.go -package=exercises_test

type exercisesRepo interface {
	Add(ctx context.Context, exercise Exercise) (*Exercise, error)
	Get(ctx context.Context, id int) (*Exercise, error)
	List(ctx context.Context, params ListParams) (_ []Exercise, total int, err error)
	ListAll(ctx context.Context, params ExerciseParams) (_ []Exercise, err error)
	Update(ctx context.Context, exercise *Exercise) error
	Delete(ctx context.Context, id int) error
	ExercisesCount(ctx context.Context, params ListParams) (int, error)
	GetExerciseTypes(ctx context.Context, params GetExerciseTypesParams) (_ []ExerciseType, err error)
}

type DeleteExerciseResponse struct {
	DeletedID int `json:"deletedId"`
}

type UpdateExerciseResponse struct {
	UpdatedID int `json:"updatedId"`
}

type AddExerciseResponse struct {
	Exercise
	CountToday int `json:"countToday"`
}

type ListResponse struct {
	Exercises []Exercise `json:"exercises"`
	Total     int        `json:"total"`
}

type Handler struct {
	repo     exercisesRepo
	analyzer *Analyzer
}

func NewHandler(repo exercisesRepo) *Handler {
	return &Handler{
		repo:     repo,
		analyzer: NewAnalyzer(repo),
	}
}

func (handler *Handler) HandleAdd(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.new")
	defer span.End()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var exercise Exercise
	if err := json.NewDecoder(r.Body).Decode(&exercise); err != nil {
		log.Tracef("new exercise, unmarshal json params: %s", err)
		http.Error(w, "add exercise failed", http.StatusBadRequest)
		return
	}

	if exercise.ExerciseID == "" || exercise.MuscleGroup == "" {
		http.Error(w, "error, exercise id or muscle group empty", http.StatusBadRequest)
		return
	}

	if exercise.CreatedAt.IsZero() {
		exercise.CreatedAt = time.Now()
	}

	addedExercise, err := handler.repo.Add(ctx, exercise)
	if err != nil {
		log.Errorf("failed to add new exercise [%s], [%s]: %s", exercise.MuscleGroup, exercise.ExerciseID, err)
		http.Error(w, "error, failed to add new exercise", http.StatusInternalServerError)
		return
	}

	todayMidnight := time.Now().Truncate(24 * time.Hour)
	tomorrowMidnight := todayMidnight.Add(24 * time.Hour)
	exercisesToday, err := handler.repo.ListAll(ctx, ExerciseParams{
		ExerciseID:         addedExercise.ExerciseID,
		MuscleGroup:        addedExercise.MuscleGroup,
		From:               &todayMidnight,
		To:                 &tomorrowMidnight,
		OnlyProd:           true,
		ExcludeTestingData: true,
	})
	if err != nil {
		// just log the error, no need to return error to the client
		log.Errorf("failed to get exercises today [%s] [%s]: %s", addedExercise.ExerciseID, addedExercise.MuscleGroup, err)
	}

	addExerciseResponse := AddExerciseResponse{
		Exercise:   *addedExercise,
		CountToday: len(exercisesToday),
	}

	addedExJson, err := json.Marshal(addExerciseResponse)
	if err != nil {
		log.Errorf("failed to marshal new exercise: %s", err)
		http.Error(w, "error, failed to add new exercise", http.StatusInternalServerError)
		return
	}

	log.Debugf("new exercise added: %s", addedExJson)
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, addedExJson, http.StatusCreated)
}

func (handler *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.get")
	defer span.End()

	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "error, id NaN", http.StatusBadRequest)
		return
	}

	e, err := handler.repo.Get(ctx, id)
	if err != nil {
		log.Errorf("failed to get exercise %d: %s", id, err)
		http.Error(w, "exercise not found", http.StatusBadRequest)
		return
	}

	exJson, err := json.Marshal(e)
	if err != nil {
		log.Errorf("failed to marshal exercise: %s", err)
		http.Error(w, "failed to marshal exercise", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, exJson, http.StatusOK)
}

func (handler *Handler) HandleExerciseHistory(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.new")
	defer span.End()

	vars := mux.Vars(r)
	exerciseID := vars["exid"]
	if exerciseID == "" {
		http.Error(w, "error, exercise id empty", http.StatusBadRequest)
		return
	}
	muscleGroup := vars["mgroup"]
	if muscleGroup == "" {
		http.Error(w, "error, muscle group empty", http.StatusBadRequest)
		return
	}

	onlyProd := false
	if r.URL.Query().Get("only_prod") == "true" {
		onlyProd = true
	}
	excludeTestingData := false
	if r.URL.Query().Get("exclude_testing_data") == "true" {
		excludeTestingData = true
	}

	exHistory, err := handler.analyzer.ExerciseHistory(ctx, ExerciseParams{
		ExerciseID:         exerciseID,
		MuscleGroup:        muscleGroup,
		OnlyProd:           onlyProd,
		ExcludeTestingData: excludeTestingData,
	})
	if err != nil {
		log.Errorf("failed to get exercise history [%s] [%s]: %s", exerciseID, muscleGroup, err)
		http.Error(w, "exercise history not found", http.StatusBadRequest)
		return
	}

	exHistoryJson, err := json.Marshal(exHistory)
	if err != nil {
		log.Errorf("failed to marshal exercise history: %s", err)
		http.Error(w, "failed to marshal exercise history", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, exHistoryJson, http.StatusOK)
}

func (handler *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.delete")
	defer span.End()

	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "error, id NaN", http.StatusBadRequest)
		return
	}

	exercise, err := handler.repo.Get(ctx, id)
	if err != nil && !errors.Is(err, ErrExerciseNotFound) {
		log.Errorf("failed to get exercise %d: %s", id, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if errors.Is(err, ErrExerciseNotFound) {
		log.Debugf("exercise %d not found", id)
		http.Error(w, "exercise not found", http.StatusNotFound)
		return
	}

	log.Debugf("deleting exercise %+v", exercise)

	if err := handler.repo.Delete(ctx, id); err != nil {
		log.Errorf("failed to delete exercise %d: %s", id, err)
		http.Error(w, "exercise not deleted", http.StatusInternalServerError)
		return
	}

	deleteRespJson, err := json.Marshal(DeleteExerciseResponse{
		DeletedID: id,
	})
	if err != nil {
		log.Errorf("failed to marshal delete response: %s", err)
		http.Error(w, "failed to marshal delete response", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(deleteRespJson))
}

func (handler *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.list")
	defer span.End()

	vars := mux.Vars(r)
	pageStr := vars["page"]
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Tracef("handle get exercises page, from <page> param: %s", err)
		http.Error(w, "parse form error, parameter <page>", http.StatusBadRequest)
		return
	}
	sizeStr := vars["size"]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		log.Tracef("handle get exercises page, from <size> param: %s", err)
		http.Error(w, "parse form error, parameter <size>", http.StatusBadRequest)
		return
	}

	if page < 1 {
		http.Error(w, "invalid page size (has to be non-zero value)", http.StatusBadRequest)
		return
	}
	if size < 1 {
		http.Error(w, "invalid size (has to be non-zero value)", http.StatusBadRequest)
		return
	}

	onlyProd := false
	onlyProdStr := r.URL.Query().Get("only_prod")
	if onlyProdStr != "" {
		onlyProd, err = strconv.ParseBool(onlyProdStr)
		if err != nil {
			log.Errorf("failed to parse onlyProd param: %s", err)
			http.Error(w, "failed to parse onlyProd param", http.StatusBadRequest)
			return
		}
	}

	excludeTestingData := false
	excludeTestingDataStr := r.URL.Query().Get("exclude_testing_data")
	if excludeTestingDataStr != "" {
		excludeTestingData, err = strconv.ParseBool(excludeTestingDataStr)
		if err != nil {
			log.Errorf("failed to parse noTesting param: %s", err)
			http.Error(w, "failed to parse noTesting param", http.StatusBadRequest)
			return
		}
	}

	listParams := ListParams{
		ExerciseParams: ExerciseParams{
			MuscleGroup:        r.URL.Query().Get("group"),
			ExerciseID:         r.URL.Query().Get("exercise_id"),
			OnlyProd:           onlyProd,
			ExcludeTestingData: excludeTestingData,
		},
		Page: page,
		Size: size,
	}

	exercises, total, err := handler.repo.List(ctx, listParams)
	if err != nil {
		log.Errorf("list exercises error: %s", err)
		http.Error(w, "failed to get exercises", http.StatusInternalServerError)
		return
	}

	exercisesPageResponse := ListResponse{
		Exercises: exercises,
		Total:     total,
	}

	exercisesPageResponseJson, err := json.Marshal(exercisesPageResponse)
	if err != nil {
		log.Errorf("marshal exercises error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, exercisesPageResponseJson, http.StatusOK)
}

func (handler *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.update")
	defer span.End()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var exercise Exercise
	if err := json.NewDecoder(r.Body).Decode(&exercise); err != nil {
		log.Errorf("update exercise, unmarshal json params: %s", err)
		http.Error(w, "update exercise failed", http.StatusBadRequest)
		return
	}

	if exercise.ExerciseID == "" || exercise.MuscleGroup == "" {
		http.Error(w, "error, exercise id or muscle group empty", http.StatusBadRequest)
		return
	}

	currentExercise, err := handler.repo.Get(ctx, exercise.ID)
	if err != nil && errors.Is(err, ErrExerciseNotFound) {
		log.Errorf("failed to get exercise %d: %s", exercise.ID, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	} else if errors.Is(err, ErrExerciseNotFound) {
		log.Debugf("exercise %d not found", exercise.ID)
		http.Error(w, "exercise not found", http.StatusNotFound)
		return
	}
	log.Debugf("update exercise %+v -> %+v", currentExercise, exercise)

	if err := handler.repo.Update(ctx, &exercise); err != nil {
		log.Errorf("failed to update exercise [%d], [%s]: %s", exercise.ID, exercise.ExerciseID, err)
		http.Error(w, "error, failed to update exercise", http.StatusInternalServerError)
		return
	}

	updateRespJson, err := json.Marshal(UpdateExerciseResponse{
		UpdatedID: exercise.ID,
	})
	if err != nil {
		log.Errorf("failed to marshal update response: %s", err)
		http.Error(w, "failed to marshal update response", http.StatusInternalServerError)
		return
	}

	log.Debugf("exercise updated: [%s] [%s]: %d", exercise.MuscleGroup, exercise.ExerciseID, exercise.ID)
	pkg.WriteJSONResponseOK(w, string(updateRespJson))
}

func (handler *Handler) HandleAvgDurationBetweenExerciseSets(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.avg-wait")
	defer span.End()

	onlyProd := false
	if r.URL.Query().Get("only_prod") == "true" {
		onlyProd = true
	}
	excludeTestingData := false
	if r.URL.Query().Get("exclude_testing_data") == "true" {
		excludeTestingData = true
	}

	avgDurationResp, err := handler.analyzer.AvgSetDuration(ctx, ExerciseParams{
		OnlyProd:           onlyProd,
		ExcludeTestingData: excludeTestingData,
	})
	if err != nil {
		log.Errorf("failed to get avg set duration between exercises: %s", err)
		http.Error(w, "failed to get avg set duration between exercises", http.StatusInternalServerError)
		return
	}

	avgDurationRespJson, err := json.Marshal(avgDurationResp)
	if err != nil {
		log.Errorf("failed to marshal avg set duration response: %s", err)
		http.Error(w, "failed to marshal avg set duration response", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, avgDurationRespJson, http.StatusOK)
}

func (handler *Handler) HandleExercisesPercentages(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.exercises-percentages")
	defer span.End()

	onlyProd := false
	if r.URL.Query().Get("only_prod") == "true" {
		onlyProd = true
	}
	excludeTestingData := false
	if r.URL.Query().Get("exclude_testing_data") == "true" {
		excludeTestingData = true
	}

	vars := mux.Vars(r)
	muscleGroup := vars["mgroup"]
	if muscleGroup == "" {
		http.Error(w, "error, muscle group empty", http.StatusBadRequest)
		return
	}

	percentages, err := handler.analyzer.ExercisePercentages(ctx, muscleGroup, onlyProd, excludeTestingData)
	if err != nil {
		log.Errorf("failed to get exercises percentages: %s", err)
		http.Error(w, "failed to get exercises percentages", http.StatusInternalServerError)
		return
	}

	percentagesJson, err := json.Marshal(percentages)
	if err != nil {
		log.Errorf("failed to marshal exercises percentages response: %s", err)
		http.Error(w, "failed to marshal exercises percentages response", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, percentagesJson, http.StatusOK)
}
