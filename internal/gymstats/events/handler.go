package events

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

//go:generate mockgen -source=$GOFILE -destination=mocks_test.go -package=events_test

type service interface {
	List(ctx context.Context, params ListParams) ([]*Event, error)
	Count(ctx context.Context, params EventParams) (int, error)
	AddTrainingStart(ctx context.Context, trainingStart TrainingStart) (int, error)
	AddTrainingFinish(ctx context.Context, trainingFinish TrainingFinish) (int, error)
	AddWeightReport(ctx context.Context, weightReport WeightReport) (int, error)
	AddPainReport(ctx context.Context, painReport PainReport) (int, error)
}

type Handler struct {
	service service
}

func NewHandler(service service) *Handler {
	return &Handler{
		service: service,
	}
}

type ListResponse struct {
	Events []*Event `json:"events"`
	Total  int      `json:"total"`
}

func (h *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.list.events")
	defer span.End()

	vars := mux.Vars(r)

	pageStr := vars["page"]
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Errorf("handle get events page, from <page> param [%s]: %s", pageStr, err)
		http.Error(w, "parse form error, parameter <page>", http.StatusBadRequest)
		return
	}
	sizeStr := vars["size"]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		log.Errorf("handle get events page, from <size> param [%s]: %s", sizeStr, err)
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

	var eventType *EventType = nil
	eventTypeParam := EventType(r.URL.Query().Get("type"))
	if eventTypeParam != "" && !eventTypeParam.IsValid() {
		log.Errorf("invalid event type: %s", eventTypeParam)
		http.Error(w, "invalid event type", http.StatusBadRequest)
		return
	} else if eventTypeParam != "" {
		eventType = &eventTypeParam
	}

	listParams := ListParams{
		EventParams: EventParams{
			Type:               eventType,
			OnlyProd:           onlyProd,
			ExcludeTestingData: excludeTestingData,

			// TODO: implement filtering by date
			// From:               nil,
			// To:                 nil,
		},
		Page: page,
		Size: size,
	}

	log.Tracef(
		"list events - page %s size %s, type [%s], only prod [%t], exclude testing data [%t]",
		pageStr, sizeStr, listParams.Type, listParams.OnlyProd, listParams.ExcludeTestingData,
	)

	events, err := h.service.List(ctx, listParams)
	if err != nil {
		log.Errorf("list events error: %s", err)
		http.Error(w, "failed to get events", http.StatusInternalServerError)
		return
	}

	total, err := h.service.Count(ctx, listParams.EventParams)
	if err != nil {
		log.Errorf("count events error: %s", err)
		http.Error(w, "failed to count events", http.StatusInternalServerError)
		return
	}

	eventsResp := ListResponse{
		Events: events,
		Total:  total,
	}

	eventsPageResponseJson, err := json.Marshal(eventsResp)
	if err != nil {
		log.Errorf("marshal events error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, eventsPageResponseJson, http.StatusOK)
}

func (h *Handler) HandleTrainingStart(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.new.trainingstart")
	defer span.End()

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
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

	trainingStartJson, err := json.Marshal(trainingStart)
	if err != nil {
		log.Errorf("failed to marshal new training start: %s", err)
		http.Error(w, "error, failed to add new training start", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, trainingStartJson, http.StatusCreated)
}

func (h *Handler) HandleTrainingFinished(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.new.trainingend")
	defer span.End()

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var trainingFinish TrainingFinish
	if err := json.NewDecoder(r.Body).Decode(&trainingFinish); err != nil {
		log.Errorf("new training finish, unmarshal json params: %s", err)
		http.Error(w, "add training finish failed", http.StatusBadRequest)
		return
	}

	id, err := h.service.AddTrainingFinish(ctx, trainingFinish)
	if err != nil {
		log.Errorf("new training finish: %s", err)
		http.Error(w, "add training finish failed", http.StatusInternalServerError)
		return
	}
	trainingFinish.ID = id

	trainingFinishJson, err := json.Marshal(trainingFinish)
	if err != nil {
		log.Errorf("failed to marshal new training finish: %s", err)
		http.Error(w, "error, failed to add new training finish", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, trainingFinishJson, http.StatusCreated)
}

func (h *Handler) HandleWeightReport(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.new.weightreport")
	defer span.End()

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var weightReport WeightReport
	if err := json.NewDecoder(r.Body).Decode(&weightReport); err != nil {
		log.Errorf("new weight report, unmarshal json params: %s", err)
		http.Error(w, "add weight report failed", http.StatusBadRequest)
		return
	}

	id, err := h.service.AddWeightReport(ctx, weightReport)
	if err != nil {
		log.Errorf("new weight report: %s", err)
		http.Error(w, "add weight report failed", http.StatusInternalServerError)
		return
	}
	weightReport.ID = id

	weightReportJson, err := json.Marshal(weightReport)
	if err != nil {
		log.Errorf("failed to marshal new weight report: %s", err)
		http.Error(w, "error, failed to add new weight report", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, weightReportJson, http.StatusCreated)
}

func (h *Handler) HandlePainReport(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "handler.gymstats.new.painreport")
	defer span.End()

	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "invalid content type", http.StatusBadRequest)
		return
	}

	var painReport PainReport
	if err := json.NewDecoder(r.Body).Decode(&painReport); err != nil {
		log.Errorf("new pain report, unmarshal json params: %s", err)
		http.Error(w, "add pain report failed", http.StatusBadRequest)
		return
	}

	id, err := h.service.AddPainReport(ctx, painReport)
	if err != nil {
		log.Errorf("new pain report: %s", err)
		http.Error(w, "add pain report failed", http.StatusInternalServerError)
		return
	}
	painReport.ID = id

	painReportJson, err := json.Marshal(painReport)
	if err != nil {
		log.Errorf("failed to marshal new pain report: %s", err)
		http.Error(w, "error, failed to add new pain report", http.StatusInternalServerError)
		return
	}
	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, painReportJson, http.StatusCreated)
}
