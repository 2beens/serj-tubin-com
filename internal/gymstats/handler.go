package gymstats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type exercisesRepo interface {
	Add(ctx context.Context, exercise *Exercise) (*Exercise, error)
	List(ctx context.Context, params ListParams) ([]Exercise, error)
	Delete(ctx context.Context, id int) error
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

func (handler *Handler) HandleDelete(w http.ResponseWriter, r *http.Request) {
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

	if err := handler.repo.Delete(r.Context(), id); err != nil {
		log.Errorf("failed to delete exercise %d: %s", id, err)
		http.Error(w, "exercise not deleted", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, fmt.Sprintf(`{"deleted":%d}`, id))
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

	if len(exercises) == 0 {
		exercises = []Exercise{}
	}

	exercisesJson, err := json.Marshal(exercises)
	if err != nil {
		log.Errorf("marshal exercises error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(exercisesJson))
}
