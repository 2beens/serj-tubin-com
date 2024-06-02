package notes_box

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type NotesListResponse struct {
	Notes []Note `json:"notes"`
	Total int    `json:"total"` // TODO: check needed
}

var _ notesRepo = (*Repo)(nil)

var _ notesRepo = (*repoMock)(nil)

type notesRepo interface {
	Add(ctx context.Context, note *Note) (*Note, error)
	Update(ctx context.Context, note *Note) error
	Get(ctx context.Context, id int) (*Note, error)
	Delete(ctx context.Context, id int) error
	List(ctx context.Context) ([]Note, error)
}

type Handler struct {
	repo    notesRepo
	metrics *metrics.Manager
}

func NewHandler(
	repo notesRepo,
	metrics *metrics.Manager,
) *Handler {
	return &Handler{
		repo:    repo,
		metrics: metrics,
	}
}

func (handler *Handler) HandleAdd(w http.ResponseWriter, r *http.Request) {
	type newNoteRequest struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	var newNoteReq newNoteRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&newNoteReq); err != nil {
			log.Errorf("new note, unmarshal json params: %s", err)
			http.Error(w, "add note failed", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("add new note failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}
		newNoteReq = newNoteRequest{
			Title:   r.Form.Get("title"),
			Content: r.Form.Get("content"),
		}
	}

	if newNoteReq.Content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}

	note := &Note{
		Title:     newNoteReq.Title,
		Content:   newNoteReq.Content,
		CreatedAt: time.Now(),
	}

	addedNote, err := handler.repo.Add(r.Context(), note)
	if err != nil {
		log.Errorf("failed to add new note [%s], [%s]: %s", note.CreatedAt, note.Title, err)
		http.Error(w, "error, failed to add new note", http.StatusInternalServerError)
		return
	}

	handler.metrics.CounterNotes.Inc()

	log.Debugf("new note added: [%s] [%s]: %d", addedNote.Title, addedNote.CreatedAt, addedNote.ID)
	pkg.WriteResponse(w, pkg.ContentType.Text, fmt.Sprintf("added:%d", addedNote.ID), http.StatusCreated)
}

func (handler *Handler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	type updateNoteRequest struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	var updateNoteReq updateNoteRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&updateNoteReq); err != nil {
			log.Errorf("update note, unmarshal json params: %s", err)
			http.Error(w, "update note failed", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("update note failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
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

		updateNoteReq = updateNoteRequest{
			ID:      id,
			Title:   r.Form.Get("title"),
			Content: r.Form.Get("content"),
		}
	}

	if updateNoteReq.Content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}

	note := &Note{
		ID:      updateNoteReq.ID,
		Title:   updateNoteReq.Title,
		Content: updateNoteReq.Content,
	}

	if err := handler.repo.Update(r.Context(), note); err != nil {
		log.Errorf("failed to update note [%d], [%s]: %s", note.ID, note.Title, err)
		http.Error(w, "error, failed to update note", http.StatusInternalServerError)
		return
	}

	log.Debugf("note updated: [%s] [%s]: %d", note.Title, note.CreatedAt, note.ID)
	pkg.WriteTextResponseOK(w, fmt.Sprintf("updated:%d", note.ID))
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

	if err := handler.repo.Delete(r.Context(), id); err != nil {
		log.Errorf("failed to delete note %d: %s", id, err)
		http.Error(w, "error, note not deleted, internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteTextResponseOK(w, fmt.Sprintf("deleted:%d", id))
}

func (handler *Handler) HandleList(w http.ResponseWriter, r *http.Request) {
	notes, err := handler.repo.List(r.Context())
	if err != nil {
		log.Errorf("list notes error: %s", err)
		http.Error(w, "failed to get notes", http.StatusInternalServerError)
		return
	}

	if len(notes) == 0 {
		notes = []Note{}
	}

	notesListRes := NotesListResponse{
		Notes: notes,
		Total: len(notes),
	}

	notesListResJson, err := json.Marshal(notesListRes)
	if err != nil {
		log.Errorf("marshal notes error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, pkg.ContentType.JSON, notesListResJson, http.StatusOK)
}
