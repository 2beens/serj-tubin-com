package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	// TODO: maybe try logging from uber
	// https://github.com/uber-go/zap
)

type BoardHandler struct {
	board *Board
}

func NewBoardHandler(boardRouter *mux.Router, board *Board) *BoardHandler {
	handler := &BoardHandler{
		board: board,
	}

	boardRouter.HandleFunc("/messages/new", handler.handleNewMessage).Methods("POST", "OPTIONS")
	boardRouter.HandleFunc("/messages/count", handler.handleMessagesCount).Methods("GET")
	boardRouter.HandleFunc("/messages/all", handler.handleGetAllMessages).Methods("GET")
	boardRouter.HandleFunc("/messages/last/{limit}", handler.handleGetAllMessages).Methods("GET")

	return handler
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

	err = handler.board.StoreMessage(boardMessage)
	if err != nil {
		log.Errorf("store new message error: %s", err)
		http.Error(w, "failed to store message", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("added"))
}

func (handler *BoardHandler) handleMessagesCount(w http.ResponseWriter, r *http.Request) {
	count, err := handler.board.MessagesCount()
	if err != nil {
		log.Errorf("get all messages count error: %s", err)
		http.Error(w, "failed to get messages count", http.StatusInternalServerError)
		return
	}
	resp := fmt.Sprintf(`{"count":%d}`, count)
	w.Write([]byte(resp))
}

func (handler *BoardHandler) handleGetAllMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	limit := -1
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

	allBboardMessages, err := handler.board.AllMessagesCache(true)
	if err != nil {
		log.Errorf("get all messages error: %s", err)
		http.Error(w, "failed to get all messages", http.StatusBadRequest)
		return
	}

	var boardMessages []*BoardMessage
	if limit == 0 || limit >= len(allBboardMessages) {
		boardMessages = allBboardMessages
	} else {
		msgCount := len(allBboardMessages)
		for i := 0; i < limit; i++ {
			boardMessages = append(boardMessages, allBboardMessages[msgCount-1-i])
		}
	}

	if len(boardMessages) == 0 {
		w.Write([]byte("[]"))
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal all messages error: %s", err)
		http.Error(w, "marshal messages error", http.StatusInternalServerError)
		return
	}

	w.Write(messagesJson)
}
