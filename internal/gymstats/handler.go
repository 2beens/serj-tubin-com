package gymstats

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type exercisesRepo interface {
	Add(ctx context.Context, exercise *Exercise) (*Exercise, error)
	Get(ctx context.Context, id int) (*Exercise, error)
	List(ctx context.Context, params ListParams) ([]Exercise, error)
	Update(ctx context.Context, exercise *Exercise) error
	Delete(ctx context.Context, id int) error
}

type DeleteExerciseResponse struct {
	DeletedID int `json:"deletedId"`
}

type UpdateExerciseResponse struct {
	UpdatedID int `json:"updatedId"`
}

type Handler struct {
	repo exercisesRepo
}

func NewHandler(repo exercisesRepo) *Handler {
	return &Handler{
		repo: repo,
	}
}

func (handler *Handler) HandleAdd(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "gymStatsHandler.new")
	defer span.End()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var exercise Exercise
	if err := json.NewDecoder(r.Body).Decode(&exercise); err != nil {
		log.Errorf("new exercise, unmarshal json params: %s", err)
		http.Error(w, "add exercise failed", http.StatusBadRequest)
		return
	}

	if exercise.ExerciseID == "" || exercise.MuscleGroup == "" {
		http.Error(w, "error, exercise id or muscle group empty", http.StatusBadRequest)
		return
	}

	addedExercise, err := handler.repo.Add(ctx, &exercise)
	if err != nil {
		log.Errorf("failed to add new exercise [%s], [%s]: %s", exercise.MuscleGroup, exercise.ExerciseID, err)
		http.Error(w, "error, failed to add new exercise", http.StatusInternalServerError)
		return
	}

	log.Debugf("new exercise added: [%s] [%s]: %d", addedExercise.MuscleGroup, addedExercise.ExerciseID, addedExercise.ID)

	addedExJson, err := json.Marshal(addedExercise)
	if err != nil {
		log.Errorf("failed to marshal new exercise: %s", err)
		http.Error(w, "error, failed to add new exercise", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, addedExJson, http.StatusCreated)
}

func (handler *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "gymStatsHandler.get")
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

func (handler *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "gymStatsHandler.delete")
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
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "gymStatsHandler.list")
	defer span.End()

	exercises, err := handler.repo.List(ctx, ListParams{
		Limit: 50,
	})
	if err != nil {
		log.Errorf("list exercises error: %s", err)
		http.Error(w, "failed to get exercises", http.StatusInternalServerError)
		return
	}

	exercisesJson, err := json.Marshal(exercises)
	if err != nil {
		log.Errorf("marshal exercises error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(exercisesJson))
}

func (handler *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "gymStatsHandler.update")
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
