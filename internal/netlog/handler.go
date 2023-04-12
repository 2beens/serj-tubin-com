package netlog

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	"go.opentelemetry.io/otel/codes"
)

type Handler struct {
	browserRequestsSecret string
	netlogApi             Api
	loginChecker          *auth.LoginChecker
	metrics               *metrics.Manager
}

func NewHandler(
	netlogApi Api,
	instrumentation *metrics.Manager,
	browserRequestsSecret string,
	loginChecker *auth.LoginChecker,
) *Handler {
	return &Handler{
		netlogApi:             netlogApi,
		metrics:               instrumentation,
		browserRequestsSecret: browserRequestsSecret,
		loginChecker:          loginChecker,
	}
}

func (handler *Handler) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/new", handler.handleNewVisit).Methods("POST", "OPTIONS").Name("new-visit")
	router.HandleFunc("/", handler.handleGetAll).Methods("GET", "OPTIONS").Name("get-last")
	router.HandleFunc("/limit/{limit}", handler.handleGetAll).Methods("GET", "OPTIONS").Name("get-with-limit")
	router.HandleFunc("/s/{source}/f/{field}/page/{page}/size/{size}", handler.handleGetPage).Methods("GET", "OPTIONS").Name("visits-page")
	router.HandleFunc("/s/{source}/f/{field}/search/{keywords}/page/{page}/size/{size}", handler.handleGetPage).Methods("GET", "OPTIONS").Name("search-page")

	router.Use(handler.authMiddleware())
}

func (handler *Handler) handleGetPage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "netlogHandler.getPage")
	defer span.End()

	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

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

	visits, err := handler.netlogApi.GetVisitsPage(ctx, keywords, field, source, page, size)
	if err != nil {
		log.Errorf("get visits error: %s", err)
		http.Error(w, "failed to get netlog visits", http.StatusInternalServerError)
		return
	}

	if len(visits) == 0 {
		resJson := fmt.Sprintf(`{"visits": %s, "total": 0}`, "[]")
		pkg.WriteResponseBytes(w, "application/json", []byte(resJson))
		return
	}

	visitsJson, err := json.Marshal(visits)
	if err != nil {
		log.Errorf("marshal netlog visits error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	allVisitsCount, err := handler.netlogApi.Count(ctx, keywords, field, source)
	if err != nil {
		log.Errorf("get netlog visits error: %s", err)
		http.Error(w, "failed to get netlog visits", http.StatusInternalServerError)
		return
	}

	resJson := fmt.Sprintf(`{"visits": %s, "total": %d}`, visitsJson, allVisitsCount)
	pkg.WriteResponseBytes(w, "application/json", []byte(resJson))
}

func (handler *Handler) handleNewVisit(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "netlogHandler.new")
	defer span.End()

	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Errorf("add new netlog visit failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	title := r.Form.Get("title")
	source := r.Form.Get("source")
	device := r.Form.Get("device")
	url := r.Form.Get("url")
	if url == "" {
		http.Error(w, "error, url empty", http.StatusBadRequest)
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

	parsedURL, err := netUrl.Parse(url)
	if err != nil {
		log.Printf("failed to parse visit url: %s", err)
		span.SetAttributes(attribute.String("visit.hostname", "<invalid/errored>"))
	} else {
		span.SetAttributes(attribute.String("visit.hostname", parsedURL.Host))
	}

	visit := &Visit{
		Title:     title,
		URL:       url,
		Source:    source,
		Device:    device,
		Timestamp: time.Unix(timestamp/1000, 0),
	}
	if err := handler.netlogApi.AddVisit(ctx, visit); err != nil {
		log.Errorf("add new visit [%s], [%s]: %s", visit.Timestamp, url, err)
		http.Error(w, "error, failed to add new visit", http.StatusInternalServerError)
		span.RecordError(err)
		return
	}

	handler.metrics.CounterNetlogVisits.Inc()

	log.Printf("new visit added: [%s] [%s]: %s", source, visit.Timestamp, visit.URL)
	// TODO: no need to return any response, status code should be enough
	pkg.WriteResponse(w, "", "added")
}

func (handler *Handler) handleGetAll(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "netlogHandler.getPage")
	defer span.End()

	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

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

	visits, err := handler.netlogApi.GetVisits(ctx, []string{}, "url", "all", limit)
	if err != nil {
		log.Errorf("get all visits error: %s", err)
		http.Error(w, "failed to get all visits", http.StatusInternalServerError)
		return
	}

	if len(visits) == 0 {
		pkg.WriteResponseBytes(w, "application/json", []byte("[]"))
		return
	}

	visitsJson, err := json.Marshal(visits)
	if err != nil {
		log.Errorf("marshal all visits error: %s", err)
		http.Error(w, "marshal all visits error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, "application/json", visitsJson)
}

func (handler *Handler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.GlobalTracer.Start(r.Context(), "netlogHandler.auth")
			defer span.End()

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				span.SetStatus(codes.Ok, "options-ok")
				return
			}

			// a non standard req. header is set, and thus - browser makes a preflight/OPTIONS request:
			//	https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#preflighted_requests
			authToken := r.Header.Get("X-SERJ-TOKEN")

			// requests coming from browser extension
			if strings.HasPrefix(r.URL.Path, "/netlog/new") {
				if handler.browserRequestsSecret != authToken {
					reqIp, _ := pkg.ReadUserIP(r)
					log.Warnf("unauthorized /netlog/new request detected from %s, authToken: %s", reqIp, authToken)
					// fool the "attacker" by a fake positive response
					pkg.WriteResponse(w, "", "added")
					span.SetStatus(codes.Error, "decoy-sent")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if authToken == "" {
				log.Tracef("[missing token] [visitor_board handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				span.SetStatus(codes.Error, "missing-auth-token")
				return
			}

			isLogged, err := handler.loginChecker.IsLogged(ctx, authToken)
			if err != nil {
				log.Tracef("[failed login check] => %s: %s", r.URL.Path, err)
				http.Error(w, "no can do", http.StatusUnauthorized)
				span.SetStatus(codes.Error, "check-logged-err")
				span.RecordError(err)
				return
			}
			if !isLogged {
				log.Tracef("[invalid token] [visitor_board handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				span.SetStatus(codes.Error, "not-logged")
				return
			}

			span.SetStatus(codes.Ok, "ok")
			next.ServeHTTP(w, r)
		})
	}
}
