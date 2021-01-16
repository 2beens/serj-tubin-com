package internal

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type NetlogHandler struct {
	visitApi     *netlog.VisitApi
	loginSession *LoginSession
}

func NewNetlogHandler(router *mux.Router, visitApi *netlog.VisitApi, loginSession *LoginSession) *NetlogHandler {
	handler := &NetlogHandler{
		visitApi:     visitApi,
		loginSession: loginSession,
	}

	router.HandleFunc("/", handler.handleGetAll).Methods("GET").Name("get-last")
	router.HandleFunc("/last/{limit}", handler.handleGetAll).Methods("GET").Name("get-last-with-limit")
	router.HandleFunc("/new", handler.handleNewVisit).Methods("POST", "OPTIONS").Name("new-visit")

	router.Use(handler.authMiddleware())

	return handler
}

func (handler *NetlogHandler) handleNewVisit(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Errorf("add new netlog visit failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	title := r.Form.Get("title")
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
		Timestamp: time.Unix(timestamp/1000, 0),
	}
	if err := handler.visitApi.AddVisit(visit); err != nil {
		log.Printf("failed to add new visit [%s], [%s]: %s", timestamp, url, err)
		http.Error(w, "error, failed to add new visit", http.StatusInternalServerError)
		return
	}

	log.Printf("new visit added: [%s]: %s", visit.Timestamp, visit.URL)
}

func (handler *NetlogHandler) handleGetAll(w http.ResponseWriter, r *http.Request) {
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

	visits, err := handler.visitApi.GetVisits(limit)
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

			//authToken := r.Header.Get("X-SERJ-TOKEN")
			//if authToken == "" || handler.loginSession.Token == "" {
			//	log.Tracef("[missing token] [board handler] unauthorized => %s", r.URL.Path)
			//	http.Error(w, "no can do", http.StatusUnauthorized)
			//	return
			//}
			//
			//if handler.loginSession.Token != authToken {
			//	log.Tracef("[invalid token] [board handler] unauthorized => %s", r.URL.Path)
			//	http.Error(w, "no can do", http.StatusUnauthorized)
			//	return
			//}

			next.ServeHTTP(w, r)
		})
	}
}
