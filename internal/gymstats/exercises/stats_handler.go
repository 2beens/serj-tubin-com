package exercises

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	exerciseIDs := r.URL.Query()["exercise_id"] // Optional: filter by exercise type(s); multi-select

	// Get progress data
	progress, err := handler.repo.GetProgressOverTime(ctx, muscleGroup, exerciseIDs)
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

// HandleProgressionRate returns progression rate comparison between current and past periods
func (handler *StatsHandler) HandleProgressionRate(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.stats.progression_rate")
	defer span.End()

	muscleGroup := r.URL.Query().Get("muscle_group")
	if muscleGroup == "" {
		muscleGroup = "all"
	}
	exerciseIDs := r.URL.Query()["exercise_id"] // Optional: filter by exercise type(s); multi-select

	daysStr := r.URL.Query().Get("days")
	if daysStr == "" {
		daysStr = "30" // Default to 30 days
	}
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		http.Error(w, "invalid days parameter (must be positive integer)", http.StatusBadRequest)
		return
	}
	// Only allow 30, 60, or 90 days
	if days != 30 && days != 60 && days != 90 {
		http.Error(w, "days parameter must be 30, 60, or 90", http.StatusBadRequest)
		return
	}

	// Get progression rate data
	progressionRate, err := handler.repo.GetProgressionRate(ctx, muscleGroup, exerciseIDs, days)
	if err != nil {
		log.Errorf("failed to get progression rate: %s", err)
		http.Error(w, "failed to get progression rate data", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"progression_rate": progressionRate,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		log.Errorf("failed to marshal progression rate response: %s", err)
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(responseJson))
}

// HandleExercisesByDateRange returns exercises for a specific date or date range
func (handler *StatsHandler) HandleExercisesByDateRange(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.stats.exercises_by_date_range")
	defer span.End()

	muscleGroup := r.URL.Query().Get("muscle_group")
	if muscleGroup == "" {
		muscleGroup = "all"
	}
	exerciseIDs := r.URL.Query()["exercise_id"] // Optional: filter by exercise type(s); multi-select

	// Parse date parameters
	dateFromStr := r.URL.Query().Get("date_from")
	dateToStr := r.URL.Query().Get("date_to")

	if dateFromStr == "" {
		http.Error(w, "date_from parameter is required", http.StatusBadRequest)
		return
	}

	// Parse dates and set to start of day (midnight) in UTC to avoid timezone issues
	dateFrom, err := time.Parse("2006-01-02", dateFromStr)
	if err != nil {
		http.Error(w, "invalid date_from format (expected YYYY-MM-DD)", http.StatusBadRequest)
		return
	}
	// Set to start of day in UTC
	dateFrom = time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 0, 0, 0, 0, time.UTC)

	// If date_to is not provided, use the same day (single date selection)
	var dateTo time.Time
	if dateToStr == "" {
		// Same day, end of day (23:59:59.999)
		dateTo = time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 23, 59, 59, 999999999, time.UTC)
	} else {
		dateTo, err = time.Parse("2006-01-02", dateToStr)
		if err != nil {
			http.Error(w, "invalid date_to format (expected YYYY-MM-DD)", http.StatusBadRequest)
			return
		}
		// End of the selected day (23:59:59.999)
		dateTo = time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 23, 59, 59, 999999999, time.UTC)
	}

	// Get exercises using ListAll with date filters
	exercises, err := handler.repo.ListAll(ctx, ExerciseParams{
		ExerciseIDs:        exerciseIDs,
		MuscleGroup:        muscleGroup,
		From:               &dateFrom,
		To:                 &dateTo,
		OnlyProd:           true,
		ExcludeTestingData: true,
	})
	if err != nil {
		log.Errorf("failed to get exercises by date range: %s", err)
		http.Error(w, "failed to get exercises", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"exercises": exercises,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		log.Errorf("failed to marshal exercises response: %s", err)
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(responseJson))
}

