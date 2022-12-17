package board

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type Handler struct {
	boardClient  *Client
	loginChecker *auth.LoginChecker
}

func NewBoardHandler(
	board *Client,
	loginChecker *auth.LoginChecker,
) *Handler {
	return &Handler{
		boardClient:  board,
		loginChecker: loginChecker,
	}
}

func (handler *Handler) SetupRoutes(router *mux.Router) {
	router.HandleFunc("/messages/new", handler.handleNewMessage).Methods("POST", "OPTIONS").Name("new-message")
	router.HandleFunc("/messages/delete/{id}", handler.handleDeleteMessage).Methods("DELETE", "OPTIONS").Name("delete-message")
	router.HandleFunc("/messages/count", handler.handleMessagesCount).Methods("GET").Name("count-messages")
	router.HandleFunc("/messages/all", handler.handleGetAllMessages).Methods("GET").Name("all-messages")
	router.HandleFunc("/messages/last/{limit}", handler.handleGetAllMessages).Methods("GET").Name("last-messages")
	router.HandleFunc("/messages/from/{from}/to/{to}", handler.handleMessagesRange).Methods("GET").Name("messages-range")
	router.HandleFunc("/messages/page/{page}/size/{size}", handler.handleGetMessagesPage).Methods("GET").Name("messages-page")

	router.Use(handler.authMiddleware())
}

func (handler *Handler) handleGetMessagesPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctx := r.Context()

	// TODO: return JSON responses for errors too (or better, check accept-content header)
	// in all handlers!

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
		http.Error(w, "invalid page size (has to be non-zero value)", http.StatusInternalServerError)
		return
	}
	if size < 1 {
		http.Error(w, "invalid size (has to be non-zero value)", http.StatusInternalServerError)
		return
	}

	boardMessages, err := handler.boardClient.GetMessagesPage(ctx, page, size)
	if err != nil {
		log.Errorf("get messages error: %s", err)
		http.Error(w, "failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	if len(boardMessages) == 0 {
		pkg.WriteResponse(w, "application/json", "[]")
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal messages error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, "application/json", messagesJson)
}

func (handler *Handler) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	messageIdStr := vars["id"]
	if messageIdStr == "" {
		log.Errorf("handle delete message: received empty message id")
		http.Error(w, "message id is empty", http.StatusInternalServerError)
		return
	}

	deleted, err := handler.boardClient.DeleteMessage(messageIdStr)
	if err != nil {
		log.Errorf("handle delete message error: %s", err)
		http.Error(w, "failed to delete message", http.StatusInternalServerError)
		return
	}

	// TODO: again - return proper JSON / requested response format
	if deleted {
		pkg.WriteResponse(w, "", "true")
	} else {
		pkg.WriteResponse(w, "", "false")
	}
}

func (handler *Handler) handleMessagesRange(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	fromStr := vars["from"]
	toStr := vars["to"]
	from, err := strconv.ParseInt(fromStr, 10, 64)
	if err != nil {
		log.Errorf("handle get messages range, from <from> param: %s", err)
		http.Error(w, "parse form error, parameter <from>", http.StatusInternalServerError)
		return
	}
	to, err := strconv.ParseInt(toStr, 10, 64)
	if err != nil {
		log.Errorf("handle get messages range, from <to> param: %s", err)
		http.Error(w, "parse form error, parameter <to>", http.StatusInternalServerError)
		return
	}

	boardMessages, err := handler.boardClient.GetMessagesWithRange(from, to)
	if err != nil {
		log.Errorf("get messages error: %s", err)
		http.Error(w, "failed to get messages", http.StatusBadRequest)
		return
	}

	if len(boardMessages) == 0 {
		pkg.WriteResponse(w, "application/json", "[]")
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal messages error: %s", err)
		http.Error(w, "marshal messages error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, "application/json", messagesJson)
}

func (handler *Handler) handleNewMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Errorf("add new message failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	message := r.Form.Get("message")
	if message == "" {
		http.Error(w, "error, message empty", http.StatusBadRequest)
		return
	}
	author := r.Form.Get("author")
	if author == "" {
		author = "anon"
	}

	boardMessage := Message{
		Timestamp: time.Now().Unix(),
		Author:    author,
		Message:   message,
	}

	id, err := handler.boardClient.NewMessage(boardMessage)
	if err != nil {
		log.Errorf("store new message error: %s", err)
		http.Error(w, "failed to store message", http.StatusInternalServerError)
		return
	}

	// TODO: refactor and unify responses
	pkg.WriteResponse(w, "", fmt.Sprintf("added:%d", id))
}

func (handler *Handler) handleMessagesCount(w http.ResponseWriter, r *http.Request) {
	count, err := handler.boardClient.MessagesCount()
	if err != nil {
		log.Errorf("get all messages count error: %s", err)
		http.Error(w, "failed to get messages count", http.StatusInternalServerError)
		return
	}

	resp := fmt.Sprintf(`{"count":%d}`, count)
	// TODO: application/json is always returned, maybe add middleware which will add it to every request
	pkg.WriteResponse(w, "application/json", resp)
}

func (handler *Handler) handleGetAllMessages(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "boardHandler.all")
	defer span.End()

	vars := mux.Vars(r)
	var limit int
	limitStr := vars["limit"]
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "invalid limit provided", http.StatusBadRequest)
			return
		}
		log.Printf("getting last %d board messages ... ", limit)
	} else {
		limit = 0
		log.Print("getting all board messages ... ")
	}

	allBoardMessages, err := handler.boardClient.AllMessagesCache(ctx, true)
	if err != nil {
		log.Errorf("get all messages error: %s", err)
		http.Error(w, "failed to get all messages", http.StatusBadRequest)
		return
	}

	var boardMessages []*Message
	if limit == 0 || limit >= len(allBoardMessages) {
		boardMessages = allBoardMessages
	} else {
		msgCount := len(allBoardMessages)
		for i := limit - 1; i >= 0; i-- {
			boardMessages = append(boardMessages, allBoardMessages[msgCount-1-i])
		}
	}

	if len(boardMessages) == 0 {
		pkg.WriteResponse(w, "application/json", "[]")
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal all messages error: %s", err)
		http.Error(w, "marshal messages error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponseBytes(w, "application/json", messagesJson)
}

func (handler *Handler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				return
			}

			// for now, only path /messages/delete/ is protected
			if !strings.HasPrefix(r.URL.Path, "/messages/delete/") {
				next.ServeHTTP(w, r)
				return
			}

			authToken := r.Header.Get("X-SERJ-TOKEN")
			if authToken == "" {
				log.Tracef("[missing token] [board handler] unauthorized => %s", r.URL.Path)
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
				log.Tracef("[invalid token] [board handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
