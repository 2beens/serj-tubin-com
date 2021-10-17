package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/2beens/serjtubincom/internal/notes_box"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type NotesBoxHandler struct {
	api          notes_box.Api
	loginSession *LoginSession
	instr        *instrumentation.Instrumentation
}

func NewNotesBoxHandler(
	api notes_box.Api,
	loginSession *LoginSession,
	instrumentation *instrumentation.Instrumentation,
) *NotesBoxHandler {
	return &NotesBoxHandler{
		api:          api,
		loginSession: loginSession,
		instr:        instrumentation,
	}
}

func (h *NotesBoxHandler) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
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

	note := &notes_box.Note{
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	addedNote, err := h.api.Add(note)
	if err != nil {
		log.Printf("failed to add new note [%s], [%s]: %s", note.CreatedAt, note.Title, err)
		http.Error(w, "error, failed to add new note", http.StatusInternalServerError)
		return
	}

	h.instr.CounterNotes.Inc()

	log.Printf("new note added: [%s] [%s]: %s", addedNote.Title, addedNote.CreatedAt, addedNote.Id)
	WriteResponse(w, "", fmt.Sprintf("added:%d", addedNote.Id))
}

func (h *NotesBoxHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
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

	deleted, err := h.api.Delete(id)
	if err != nil {
		log.Printf("failed to delete note %d: %s", id, err)
		http.Error(w, "error, note not deleted, internal server error", http.StatusInternalServerError)
		return
	}

	if deleted {
		WriteResponse(w, "", fmt.Sprintf("deleted:%d", id))
	} else {
		WriteResponse(w, "", fmt.Sprintf("not-deleted:%d", id))
	}
}

func (h *NotesBoxHandler) handleList(w http.ResponseWriter, r *http.Request) {
	notes, err := h.api.List()
	if err != nil {
		log.Errorf("list notes error: %s", err)
		http.Error(w, "failed to get notes", http.StatusInternalServerError)
		return
	}

	if notes == nil || len(notes) == 0 {
		notes = []notes_box.Note{}
	}

	notesJson, err := json.Marshal(notes)
	if err != nil {
		log.Errorf("marshal notes error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resJson := fmt.Sprintf(`{"notes": %s, "total": %d}`, notesJson, len(notes))
	WriteResponseBytes(w, "application/json", []byte(resJson))
}

func (handler *NotesBoxHandler) authMiddleware() func(next http.Handler) http.Handler {
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

			if authToken == "" || handler.loginSession.Token == "" {
				log.Tracef("[missing token] [notes handler] [%s] unauthorized => %s", handler.loginSession.Token, r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}
			if authToken != handler.loginSession.Token {
				log.Tracef("[invalid token] [notes handler] ['%s' != '%s'] unauthorized => %s", authToken, handler.loginSession.Token, r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
