package visitor_board

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"
)

type Handler struct {
	repo         boardMessagesRepo
	loginChecker *auth.LoginChecker
}

var _ boardMessagesRepo = (*Repo)(nil)
var _ boardMessagesRepo = (*repoMock)(nil)

type boardMessagesRepo interface {
	Add(ctx context.Context, message Message) (int, error)
	Delete(ctx context.Context, id int) error
	List(ctx context.Context, options ...func(listOptions *ListOptions)) ([]Message, error)
	GetMessagesPage(ctx context.Context, page, size int) ([]Message, error)
	AllMessagesCount(ctx context.Context) (int, error)
}

func NewBoardHandler(
	repo boardMessagesRepo,
	loginChecker *auth.LoginChecker,
) *Handler {
	return &Handler{
		repo:         repo,
		loginChecker: loginChecker,
	}
}

func (handler *Handler) SetupRoutes(router *mux.Router) {
	// TODO: check which routes are used
	router.HandleFunc("/board/messages/new", handler.handleNewMessage).Methods("POST", "OPTIONS").Name("new-message")
	router.HandleFunc("/board/messages/delete/{id}", handler.handleDeleteMessage).Methods("DELETE", "OPTIONS").Name("delete-message")
	router.HandleFunc("/board/messages/count", handler.handleMessagesCount).Methods("GET").Name("count-messages")
	router.HandleFunc("/board/messages/all", handler.handleGetAllMessages).Methods("GET").Name("all-messages")
	router.HandleFunc("/board/messages/last/{limit}", handler.handleGetAllMessages).Methods("GET").Name("last-messages")
	router.HandleFunc("/board/messages/page/{page}/size/{size}", handler.handleGetMessagesPage).Methods("GET").Name("messages-page")
}

func (handler *Handler) handleGetMessagesPage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "boardHandler.messagesPage")
	defer span.End()

	// TODO: return JSON responses for errors too (or better, check accept-content header)
	// in all handlers!

	vars := mux.Vars(r)
	pageStr := vars["page"]
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Errorf("handle get messages page, from <page> param: %s", err)
		http.Error(w, "parse form error, parameter <page>", http.StatusBadRequest)
		return
	}
	sizeStr := vars["size"]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		log.Errorf("handle get messages page, from <size> param: %s", err)
		http.Error(w, "parse form error, parameter <size>", http.StatusInternalServerError)
		return
	}

	log.Tracef("page %s size %s", pageStr, sizeStr)

	if page < 1 {
		http.Error(w, "invalid page size (has to be non-zero value)", http.StatusBadRequest)
		return
	}
	if size < 1 {
		http.Error(w, "invalid size (has to be non-zero value)", http.StatusBadRequest)
		return
	}

	boardMessages, err := handler.repo.GetMessagesPage(ctx, page, size)
	if err != nil {
		span.RecordError(err)
		log.Errorf("get messages error: %s", err)
		http.Error(w, "failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	if len(boardMessages) == 0 {
		pkg.WriteJSONResponseOK(w, "[]")
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		span.RecordError(err)
		log.Errorf("marshal messages error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(messagesJson))
}

func (handler *Handler) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "boardHandler.delete")
	defer span.End()

	vars := mux.Vars(r)

	messageIdStr := vars["id"]
	if messageIdStr == "" {
		log.Errorf("handle delete message: received empty message id")
		http.Error(w, "message id is empty", http.StatusBadRequest)
		return
	}
	messageId, err := strconv.Atoi(messageIdStr)
	if err != nil {
		log.Errorf("handle delete message: received invalid message id")
		http.Error(w, "message id is invalid", http.StatusBadRequest)
		return
	}

	err = handler.repo.Delete(ctx, messageId)
	if err != nil {
		span.RecordError(err)
		log.Errorf("handle delete message error: %s", err)
	}

	switch {
	case errors.Is(err, ErrMessageNotFound):
		http.Error(w, "message not found", http.StatusNotFound)
	case err != nil:
		http.Error(w, "failed to delete message", http.StatusInternalServerError)
	}

	// TODO: again - return proper JSON / requested response format (i.e. here just status code)
	pkg.WriteTextResponseOK(w, "true")
}

func (handler *Handler) handleNewMessage(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "boardHandler.new")
	defer span.End()

	var message Message
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			log.Errorf("store new message, unmarshal message json params: %s", err)
			http.Error(w, "failed to store message", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			span.RecordError(err)
			log.Errorf("add new message failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}
		message = Message{
			Author:  r.Form.Get("author"),
			Message: r.Form.Get("message"),
		}
	}

	if message.Message == "" {
		http.Error(w, "error, message empty", http.StatusBadRequest)
		return
	}
	if message.Author == "" {
		message.Author = "anon"
	}

	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}

	id, err := handler.repo.Add(ctx, message)
	if err != nil {
		span.RecordError(err)
		log.Errorf("store new message error: %s", err)
		http.Error(w, "failed to store message", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponse(w, pkg.ContentType.Text, fmt.Sprintf("added:%d", id), http.StatusCreated)
}

func (handler *Handler) handleMessagesCount(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "boardHandler.count")
	defer span.End()

	count, err := handler.repo.AllMessagesCount(ctx)
	if err != nil {
		span.RecordError(err)
		log.Errorf("get all messages count error: %s", err)
		http.Error(w, "failed to get messages count", http.StatusInternalServerError)
		return
	}

	resp := fmt.Sprintf(`{"count":%d}`, count)
	pkg.WriteJSONResponseOK(w, resp)
}

func (handler *Handler) handleGetAllMessages(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "boardHandler.all")
	defer span.End()

	vars := mux.Vars(r)
	var (
		err      error
		messages []Message
	)
	if limitStr := vars["limit"]; limitStr != "" {
		limit, lErr := strconv.Atoi(limitStr)
		if lErr != nil {
			http.Error(w, "invalid limit provided", http.StatusBadRequest)
			return
		}
		span.SetAttributes(attribute.Int("limit", limit))
		messages, err = handler.repo.List(ctx, ListWithLimit(limit))
	} else {
		messages, err = handler.repo.List(ctx)
	}

	if err != nil {
		span.RecordError(err)
		log.Errorf("get all messages error: %s", err)
		http.Error(w, "failed to get all messages", http.StatusInternalServerError)
		return
	}

	span.SetAttributes(attribute.Int("found", len(messages)))

	if len(messages) == 0 {
		pkg.WriteJSONResponseOK(w, "[]")
		return
	}

	messagesJson, err := json.Marshal(messages)
	if err != nil {
		span.RecordError(err)
		log.Errorf("marshal all messages error: %s", err)
		http.Error(w, "marshal messages error", http.StatusInternalServerError)
		return
	}

	pkg.WriteJSONResponseOK(w, string(messagesJson))
}
