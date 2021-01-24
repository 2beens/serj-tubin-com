package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	// TODO: maybe try logging from uber
	// https://github.com/uber-go/zap
	log "github.com/sirupsen/logrus"
)

type BoardHandler struct {
	board        *BoardApi
	loginSession *LoginSession
}

func NewBoardHandler(boardRouter *mux.Router, board *BoardApi, loginSession *LoginSession) *BoardHandler {
	handler := &BoardHandler{
		board:        board,
		loginSession: loginSession,
	}

	router.HandleFunc("/messages/new", handler.handleNewMessage).Methods("POST", "OPTIONS").Name("new-message")
	router.HandleFunc("/messages/delete/{id}", handler.handleDeleteMessage).Methods("DELETE", "OPTIONS").Name("delete-message")
	router.HandleFunc("/messages/count", handler.handleMessagesCount).Methods("GET").Name("count-messages")
	router.HandleFunc("/messages/all", handler.handleGetAllMessages).Methods("GET").Name("all-messages")
	router.HandleFunc("/messages/last/{limit}", handler.handleGetAllMessages).Methods("GET").Name("last-messages")
	router.HandleFunc("/messages/from/{from}/to/{to}", handler.handleMessagesRange).Methods("GET").Name("messages-range")
	router.HandleFunc("/messages/page/{page}/size/{size}", handler.handleGetMessagesPage).Methods("GET").Name("messages-page")

	router.Use(handler.authMiddleware())

	return handler
}

func (handler *BoardHandler) handleGetMessagesPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

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

	boardMessages, err := handler.board.GetMessagesPage(page, size)
	if err != nil {
		log.Errorf("get messages error: %s", err)
		http.Error(w, "failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	if len(boardMessages) == 0 {
		WriteResponse(w, "application/json", "[]")
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal messages error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", messagesJson)
}

func (handler *BoardHandler) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	messageIdStr := vars["id"]
	if messageIdStr == "" {
		log.Errorf("handle delete message: received empty message id")
		http.Error(w, "message id is empty", http.StatusInternalServerError)
		return
	}

	deleted, err := handler.board.DeleteMessage(messageIdStr)
	if err != nil {
		log.Errorf("handle delete message error: %s", err)
		http.Error(w, "failed to delete message", http.StatusInternalServerError)
		return
	}

	// TODO: again - return proper JSON / requested response format
	if deleted {
		WriteResponse(w, "", "true")
	} else {
		WriteResponse(w, "", "false")
	}
}

func (handler *BoardHandler) handleMessagesRange(w http.ResponseWriter, r *http.Request) {
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

	boardMessages, err := handler.board.GetMessagesWithRange(from, to)
	if err != nil {
		log.Errorf("get messages error: %s", err)
		http.Error(w, "failed to get messages", http.StatusBadRequest)
		return
	}

	if len(boardMessages) == 0 {
		WriteResponse(w, "application/json", "[]")
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal messages error: %s", err)
		http.Error(w, "marshal messages error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", messagesJson)
}

func (handler *BoardHandler) handleNewMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
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

	boardMessage := BoardMessage{
		Timestamp: time.Now().Unix(),
		Author:    author,
		Message:   message,
	}

	id, err := handler.board.StoreMessage(boardMessage)
	if err != nil {
		log.Errorf("store new message error: %s", err)
		http.Error(w, "failed to store message", http.StatusInternalServerError)
		return
	}

	// TODO: refactor and unify responses
	WriteResponse(w, "", fmt.Sprintf("added:%d", id))
}

func (handler *BoardHandler) handleMessagesCount(w http.ResponseWriter, r *http.Request) {
	count, err := handler.board.MessagesCount()
	if err != nil {
		log.Errorf("get all messages count error: %s", err)
		http.Error(w, "failed to get messages count", http.StatusInternalServerError)
		return
	}

	resp := fmt.Sprintf(`{"count":%d}`, count)
	// TODO: application/json is always returned, maybe add middleware which will add it to every request
	WriteResponse(w, "application/json", resp)
}

func (handler *BoardHandler) handleGetAllMessages(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("getting last %d boardApi messages ... ", limit)
	} else {
		limit = 0
		log.Print("getting all boardApi messages ... ")
	}

	allBoardMessages, err := handler.board.AllMessagesCache(true)
	if err != nil {
		log.Errorf("get all messages error: %s", err)
		http.Error(w, "failed to get all messages", http.StatusBadRequest)
		return
	}

	var boardMessages []*BoardMessage
	if limit == 0 || limit >= len(allBoardMessages) {
		boardMessages = allBoardMessages
	} else {
		msgCount := len(allBoardMessages)
		for i := limit - 1; i >= 0; i-- {
			boardMessages = append(boardMessages, allBoardMessages[msgCount-1-i])
		}
	}

	if len(boardMessages) == 0 {
		WriteResponse(w, "application/json", "[]")
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal all messages error: %s", err)
		http.Error(w, "marshal messages error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", messagesJson)
}

func (handler *BoardHandler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
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
			if authToken == "" || handler.loginSession.Token == "" {
				log.Tracef("[missing token] [boardApi handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			if handler.loginSession.Token != authToken {
				log.Tracef("[invalid token] [boardApi handler] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
