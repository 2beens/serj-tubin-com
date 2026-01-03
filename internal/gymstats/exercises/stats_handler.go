package exercises

import (
	"encoding/json"
	"net/http"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	log "github.com/sirupsen/logrus"
)

type StatsHandler struct {
	repo exercisesRepo
}

func NewStatsHandler(repo exercisesRepo) *StatsHandler {
	return &StatsHandler{
		repo: repo,
	}
}

// HandleVerifyToken is a simple endpoint to verify if the auth token is valid
func (handler *StatsHandler) HandleVerifyToken(w http.ResponseWriter, r *http.Request) {
	// If we reach here, the auth middleware has already validated the token
	pkg.WriteJSONResponseOK(w, `{"status": "ok"}`)
}

// HandleProgress returns progress statistics over time for a muscle group
func (handler *StatsHandler) HandleProgress(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.stats.progress")
	defer span.End()

	muscleGroup := r.URL.Query().Get("muscle_group")
	if muscleGroup == "" {
		muscleGroup = "all"
	}
	exerciseID := r.URL.Query().Get("exercise_id") // Optional: filter by exercise type

	// Get progress data
	progress, err := handler.repo.GetProgressOverTime(ctx, muscleGroup, exerciseID)
	if err != nil {
		log.Errorf("failed to get progress over time: %s", err)
		http.Error(w, "failed to get progress data", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"progress": progress,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		log.Errorf("failed to marshal progress response: %s", err)
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(responseJson))
}

