package gymstats

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	log "github.com/sirupsen/logrus"
)

type exercisesRepo interface {
	Add(ctx context.Context, exercise *Exercise) (*Exercise, error)
	List(ctx context.Context) ([]Exercise, error)
}

type Handler struct {
	repo         exercisesRepo
	loginChecker *auth.LoginChecker
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

	// TODO: validate exercise

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

func (handler *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "gymStatsHandler.list")
	defer span.End()

	exercises, err := handler.repo.List(ctx)
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
