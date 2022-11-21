package notes_box

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/pkg"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/metrics"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	api          Api
	loginChecker *auth.LoginChecker
	metrics      *metrics.Manager
}

func NewHandler(
	api Api,
	loginChecker *auth.LoginChecker,
	metrics *metrics.Manager,
) *Handler {
	return &Handler{
		api:          api,
		loginChecker: loginChecker,
		metrics:      metrics,
	}
}

func (handler *Handler) HandleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Errorf("add new note failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	title := r.Form.Get("title")
	content := r.Form.Get("content")
	if content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}

	note := &Note{
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	addedNote, err := handler.api.Add(r.Context(), note)
	if err != nil {
		log.Printf("failed to add new note [%s], [%s]: %s", note.CreatedAt, note.Title, err)
		http.Error(w, "error, failed to add new note", http.StatusInternalServerError)
		return
	}

	handler.metrics.CounterNotes.Inc()

	log.Printf("new note added: [%s] [%s]: %d", addedNote.Title, addedNote.CreatedAt, addedNote.Id)
	pkg.WriteResponse(w, "", fmt.Sprintf("added:%d", addedNote.Id))
}

func (handler *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Add("Allow", "PUT, OPTIONS")
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Errorf("update note failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	title := r.Form.Get("title")
	content := r.Form.Get("content")
	if content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}
	idStr := r.Form.Get("id")
	if idStr == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "error, id invalid", http.StatusBadRequest)
		return
	}

	note := &Note{
		Id:      id,
		Title:   title,
		Content: content,
		// CreatedAt: not updateable for now,
	}

	if err := handler.api.Update(r.Context(), note); err != nil {
		log.Printf("failed to update note [%d], [%s]: %s", note.Id, note.Title, err)
		http.Error(w, "error, failed to update note", http.StatusInternalServerError)
		return
	}

	log.Printf("note updated: [%s] [%s]: %d", note.Title, note.CreatedAt, note.Id)
	pkg.WriteResponse(w, "", fmt.Sprintf("updated:%d", note.Id))
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

	if err := handler.api.Delete(r.Context(), id); err != nil {
		log.Printf("failed to delete note %d: %s", id, err)
		http.Error(w, "error, note not deleted, internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponse(w, "", fmt.Sprintf("deleted:%d", id))
}

func (handler *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	notes, err := handler.api.List(r.Context())
	if err != nil {
		log.Errorf("list notes error: %s", err)
		http.Error(w, "failed to get notes", http.StatusInternalServerError)
		return
	}

	if len(notes) == 0 {
		notes = []Note{}
	}

	notesJson, err := json.Marshal(notes)
	if err != nil {
		log.Errorf("marshal notes error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resJson := fmt.Sprintf(`{"notes": %s, "total": %d}`, notesJson, len(notes))
	pkg.WriteResponseBytes(w, "application/json", []byte(resJson))
}

func (handler *Handler) AuthMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				return
			}

			// a non standard req. header is set, and thus - browser makes a preflight/OPTIONS request:
			//	https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#preflighted_requests
			authToken := r.Header.Get("X-SERJ-TOKEN")

			if authToken == "" {
				log.Tracef("[missing token] [notes handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			isLogged, err := handler.loginChecker.IsLogged(r.Context(), authToken)
			if err != nil {
				log.Tracef("[failed login check] => %s: %s", r.URL.Path, err)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}
			if !isLogged {
				log.Tracef("[invalid token] [notes handler] unauthorized token %s => %s", authToken, r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
