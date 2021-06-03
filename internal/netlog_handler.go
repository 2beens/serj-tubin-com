package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type NetlogHandler struct {
	browserRequestsSecret string
	netlogApi             netlog.Api
	loginSession          *LoginSession
	instr                 *Instrumentation
}

func NewNetlogHandler(
	router *mux.Router,
	netlogApi netlog.Api,
	instrumentation *Instrumentation,
	browserRequestsSecret string,
	loginSession *LoginSession,
) *NetlogHandler {
	handler := &NetlogHandler{
		netlogApi:             netlogApi,
		instr:                 instrumentation,
		browserRequestsSecret: browserRequestsSecret,
		loginSession:          loginSession,
	}

	router.HandleFunc("/new", handler.handleNewVisit).Methods("POST", "OPTIONS").Name("new-visit")
	router.HandleFunc("/", handler.handleGetAll).Methods("GET", "OPTIONS").Name("get-last")
	router.HandleFunc("/limit/{limit}", handler.handleGetAll).Methods("GET", "OPTIONS").Name("get-with-limit")
	router.HandleFunc("/s/{source}/f/{field}/page/{page}/size/{size}", handler.handleGetPage).Methods("GET", "OPTIONS").Name("visits-page")
	router.HandleFunc("/s/{source}/f/{field}/search/{keywords}/page/{page}/size/{size}", handler.handleGetPage).Methods("GET", "OPTIONS").Name("search-page")

	router.Use(handler.authMiddleware())

	return handler
}

func (handler *NetlogHandler) handleGetPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
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

	visits, err := handler.netlogApi.GetVisitsPage(keywords, field, source, page, size)
	if err != nil {
		log.Errorf("get visits error: %s", err)
		http.Error(w, "failed to get netlog visits", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	if len(visits) == 0 {
		resJson := fmt.Sprintf(`{"visits": %s, "total": 0}`, "[]")
		WriteResponseBytes(w, "application/json", []byte(resJson))
		return
	}

	visitsJson, err := json.Marshal(visits)
	if err != nil {
		log.Errorf("marshal netlog visits error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	allVisitsCount, err := handler.netlogApi.Count(keywords, field, source)
	if err != nil {
		log.Errorf("get netlog visits error: %s", err)
		http.Error(w, "failed to get netlog visits", http.StatusInternalServerError)
		return
	}

	resJson := fmt.Sprintf(`{"visits": %s, "total": %d}`, visitsJson, allVisitsCount)
	WriteResponseBytes(w, "application/json", []byte(resJson))
}

func (handler *NetlogHandler) handleNewVisit(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
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

	visit := &netlog.Visit{
		Title:     title,
		URL:       url,
		Source:    source,
		Timestamp: time.Unix(timestamp/1000, 0),
	}
	if err := handler.netlogApi.AddVisit(visit); err != nil {
		log.Printf("failed to add new visit [%s], [%s]: %s", visit.Timestamp, url, err)
		http.Error(w, "error, failed to add new visit", http.StatusInternalServerError)
		return
	}

	handler.instr.CounterNetlogVisits.Inc()

	log.Printf("new visit added: [%s] [%s]: %s", source, visit.Timestamp, visit.URL)
	WriteResponse(w, "", "added")
}

func (handler *NetlogHandler) handleGetAll(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
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

	visits, err := handler.netlogApi.GetVisits([]string{}, "url", "all", limit)
	if err != nil {
		log.Errorf("get all visits error: %s", err)
		http.Error(w, "failed to get all visits", http.StatusInternalServerError)
		return
	}

	if len(visits) == 0 {
		WriteResponseBytes(w, "application/json", []byte("[]"))
		return
	}

	visitsJson, err := json.Marshal(visits)
	if err != nil {
		log.Errorf("marshal all visits error: %s", err)
		http.Error(w, "marshal all visits error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", visitsJson)
}

func (handler *NetlogHandler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				return
			}

			// a non standard req. header is set, and thus - browser makes a preflight/OPTIONS request:
			//	https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#preflighted_requests
			authToken := r.Header.Get("X-SERJ-TOKEN")

			// requests coming from browser extension
			if strings.HasPrefix(r.URL.Path, "/netlog/new") {
				if handler.browserRequestsSecret != authToken {
					reqIp, _ := ReadUserIP(r)
					log.Warnf("unauthorized /netlog/new request detected from %s, authToken: %s", reqIp, authToken)
					// fool the "attacker" by a fake positive response
					WriteResponse(w, "", "added")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if authToken == "" || handler.loginSession.Token == "" {
				log.Tracef("[missing token] [board handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}
			if authToken != handler.loginSession.Token {
				log.Tracef("[invalid token] [board handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
