package events

import (
	"encoding/json"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) HandleAddTrainingStart(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.new.trainingstart")
	defer span.End()

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var trainingStart TrainingStart
	if err := json.NewDecoder(r.Body).Decode(&trainingStart); err != nil {
		log.Errorf("new training start, unmarshal json params: %s", err)
		http.Error(w, "add training start failed", http.StatusBadRequest)
		return
	}

	id, err := h.service.AddTrainingStart(ctx, trainingStart)
	if err != nil {
		log.Errorf("new training start: %s", err)
		http.Error(w, "add training start failed", http.StatusInternalServerError)
		return
	}
	trainingStart.ID = id

	w.WriteHeader(http.StatusCreated)

	trainingStartJson, err := json.Marshal(trainingStart)
	if err != nil {
		log.Errorf("failed to marshal new training start: %s", err)
		http.Error(w, "error, failed to add new training start", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, trainingStartJson, http.StatusCreated)
}
