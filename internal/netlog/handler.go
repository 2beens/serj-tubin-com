package netlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	netUrl "net/url"
	"strconv"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

type netlogRepo interface {
	AddVisit(ctx context.Context, visit *Visit) error
	GetVisits(ctx context.Context, keywords []string, field string, source string, limit int) ([]*Visit, error)
	CountAll(ctx context.Context) (int, error)
	Count(ctx context.Context, keywords []string, field string, source string) (int, error)
	GetVisitsPage(ctx context.Context, keywords []string, field string, source string, page int, size int) ([]*Visit, error)
}

type newVisitRequest struct {
	Title     string `json:"title"`
	Source    string `json:"source"`
	Device    string `json:"device"`
	URL       string `json:"url"`
	Timestamp int64  `json:"timestamp"`
}

type Handler struct {
	browserRequestsSecret string
	repo                  netlogRepo
	loginChecker          *auth.LoginChecker
	metrics               *metrics.Manager
}

func NewHandler(
	repo netlogRepo,
	instrumentation *metrics.Manager,
	browserRequestsSecret string,
	loginChecker *auth.LoginChecker,
) *Handler {
	return &Handler{
		repo:                  repo,
		metrics:               instrumentation,
		browserRequestsSecret: browserRequestsSecret,
		loginChecker:          loginChecker,
	}
}

func (handler *Handler) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/netlog/new", handler.handleNewVisit).Methods("POST", "OPTIONS").Name("new-visit")
	router.HandleFunc("/netlog/", handler.handleGetAll).Methods("GET", "OPTIONS").Name("get-last")
	router.HandleFunc("/netlog/limit/{limit}", handler.handleGetAll).Methods("GET", "OPTIONS").Name("get-with-limit")
	router.HandleFunc("/netlog/s/{source}/f/{field}/page/{page}/size/{size}", handler.handleGetPage).Methods("GET", "OPTIONS").Name("visits-page")
	router.HandleFunc("/netlog/s/{source}/f/{field}/search/{keywords}/page/{page}/size/{size}", handler.handleGetPage).Methods("GET", "OPTIONS").Name("search-page")
}

func (handler *Handler) handleGetPage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "netlogHandler.getPage")
	defer span.End()

	vars := mux.Vars(r)

	field := vars["field"]
	source := vars["source"]

	pageStr := vars["page"]
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Errorf("handle get netlog visits page, from <page> param: %s", err)
		http.Error(w, "parse form error, parameter <page>", http.StatusBadRequest)
		return
	}
	sizeStr := vars["size"]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		log.Errorf("handle get netlog visits page, from <size> param: %s", err)
		http.Error(w, "parse form error, parameter <size>", http.StatusInternalServerError)
		return
	}

	var keywords []string
	keywordsRaw := vars["keywords"]
	if keywordsRaw != "" {
		keywords = strings.Split(keywordsRaw, ",")
	}

	log.Tracef("get netlog visits: s[%s], f[%s], page %s size %s, keywords: %s", source, field, pageStr, sizeStr, keywords)

	if page < 1 {
		http.Error(w, "invalid page size (has to be non-zero value)", http.StatusInternalServerError)
		return
	}
	if size < 1 {
		http.Error(w, "invalid size (has to be non-zero value)", http.StatusInternalServerError)
		return
	}

	visits, err := handler.repo.GetVisitsPage(ctx, keywords, field, source, page, size)
	if err != nil {
		log.Errorf("get visits error: %s", err)
		http.Error(w, "failed to get netlog visits", http.StatusInternalServerError)
		return
	}

	if len(visits) == 0 {
		resJson := fmt.Sprintf(`{"visits": %s, "total": 0}`, "[]")
		pkg.WriteJSONResponseOK(w, resJson)
		return
	}

	visitsJson, err := json.Marshal(visits)
	if err != nil {
		log.Errorf("marshal netlog visits error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	allVisitsCount, err := handler.repo.Count(ctx, keywords, field, source)
	if err != nil {
		log.Errorf("get netlog visits error: %s", err)
		http.Error(w, "failed to get netlog visits", http.StatusInternalServerError)
		return
	}

	resJson := fmt.Sprintf(`{"visits": %s, "total": %d}`, visitsJson, allVisitsCount)
	pkg.WriteJSONResponseOK(w, resJson)
}

func (handler *Handler) handleNewVisit(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "netlogHandler.new")
	defer span.End()

	var reqData newVisitRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			log.Errorf("add new netlog visit failed, decode json error: %s", err)
			http.Error(w, "decode json error", http.StatusInternalServerError)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("add new netlog visit failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}

		timestampStr := r.Form.Get("timestamp")
		if timestampStr == "" {
			http.Error(w, "error, timestamp empty", http.StatusBadRequest)
			return
		}
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			http.Error(w, "error, timestamp invalid", http.StatusBadRequest)
			return
		}

		reqData = newVisitRequest{
			Title:     r.Form.Get("title"),
			Source:    r.Form.Get("source"),
			Device:    r.Form.Get("device"),
			URL:       r.Form.Get("url"),
			Timestamp: timestamp,
		}
	}

	decodedURL, err := url.QueryUnescape(reqData.URL)
	if err != nil {
		log.Errorf("add new netlog visit failed, decode url error: %s", err)
		http.Error(w, "decode url error", http.StatusInternalServerError)
		return
	}
	reqData.URL = decodedURL

	decodedTitle, err := url.QueryUnescape(reqData.Title)
	if err != nil {
		log.Errorf("add new netlog visit failed, decode title error: %s", err)
		http.Error(w, "decode title error", http.StatusInternalServerError)
		return
	}
	reqData.Title = decodedTitle

	if reqData.URL == "" {
		http.Error(w, "error, url empty", http.StatusBadRequest)
		return
	}

	parsedURL, err := netUrl.Parse(reqData.URL)
	if err != nil {
		log.Errorf("failed to parse visit url: %s", err)
		span.SetAttributes(attribute.String("visit.hostname", "<invalid/errored>"))
	} else {
		span.SetAttributes(attribute.String("visit.hostname", parsedURL.Host))
	}

	span.SetAttributes(attribute.String("visit.device", reqData.Source))
	span.SetAttributes(attribute.String("visit.device", reqData.Device))

	visit := &Visit{
		Title:     reqData.Title,
		URL:       reqData.URL,
		Source:    reqData.Source,
		Device:    reqData.Device,
		Timestamp: time.Unix(reqData.Timestamp/1000, 0),
	}
	if err := handler.repo.AddVisit(ctx, visit); err != nil {
		log.Errorf("add new visit [%d], [%s] [%s]: %s", reqData.Timestamp, reqData.Source, reqData.Device, err)
		http.Error(w, "error, failed to add new visit", http.StatusInternalServerError)
		span.RecordError(err)
		return
	}

	handler.metrics.CounterNetlogVisits.Inc()

	log.WithFields(log.Fields{
		"timestamp": visit.Timestamp,
		"source":    visit.Source,
		"device":    visit.Device,
	}).Print("new visit added")

	pkg.WriteResponse(w, pkg.ContentType.Text, "added", http.StatusCreated)
}

func (handler *Handler) handleGetAll(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "netlogHandler.getPage")
	defer span.End()

	vars := mux.Vars(r)

	// TODO: maybe make configurable
	limit := 200 // default value
	if limitStr := vars["limit"]; limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid limit provided", http.StatusBadRequest)
			return
		}
	}

	log.Printf("getting last %d netlog visits ... ", limit)

	visits, err := handler.repo.GetVisits(ctx, []string{}, "url", "all", limit)
	if err != nil {
		log.Errorf("get all visits error: %s", err)
		http.Error(w, "failed to get all visits", http.StatusInternalServerError)
		return
	}

	if len(visits) == 0 {
		pkg.WriteJSONResponseOK(w, "[]")
		return
	}

	visitsJson, err := json.Marshal(visits)
	if err != nil {
		log.Errorf("marshal all visits error: %s", err)
		http.Error(w, "marshal all visits error", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(visitsJson))
}
