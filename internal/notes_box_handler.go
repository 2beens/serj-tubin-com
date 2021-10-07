package internal

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/2beens/serjtubincom/internal/notes_box"
	log "github.com/sirupsen/logrus"
)

type NotesBoxHandler struct {
	api notes_box.Api
}

func NewNotesBoxHandler(api notes_box.Api) *NotesBoxHandler {
	return &NotesBoxHandler{
		api: api,
	}
}

func (h *NotesBoxHandler) handleAdd(w http.ResponseWriter, r *http.Request) {

}

func (h *NotesBoxHandler) handleRemove(w http.ResponseWriter, r *http.Request) {

}

func (h *NotesBoxHandler) handleList(w http.ResponseWriter, r *http.Request) {
	notes, err := h.api.List()
	if err != nil {
		log.Errorf("list notes error: %s", err)
		http.Error(w, "failed to get notes", http.StatusInternalServerError)
		return
	}

	if len(notes) == 0 {
		WriteResponseBytes(w, "application/json", []byte("[]"))
		return
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
