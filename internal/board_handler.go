package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type BoardHandler struct {
	board *Board
}

func NewBoardHandler(boardRouter *mux.Router, board *Board) *BoardHandler {
	handler := &BoardHandler{
		board: board,
	}

	boardRouter.HandleFunc("/messages/new", handler.handleNewMessage).Methods("POST")
	boardRouter.HandleFunc("/messages/count", handler.handleMessagesCount).Methods("GET")
	boardRouter.HandleFunc("/messages/all", handler.handleGetAllMessages).Methods("GET")

	return handler
}

func (handler *BoardHandler) handleNewMessage(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("add new message failed, parse form error: %s", err)
		w.Write([]byte("error 500: parse form error"))
		return
	}

	message := r.Form.Get("message")
	if message == "" {
		w.Write([]byte("error, message empty"))
		return
	}
	author := r.Form.Get("author")

	boardMessage := BoardMessage{
		Timestamp: time.Now().Unix(),
		Author:    author,
		Message:   message,
	}

	err = handler.board.StoreMessage(boardMessage)

	if err != nil {
		log.Errorf("store new message error: %s", err)
		w.Write([]byte("error 500: get messages error"))
		return
	}

	w.Write([]byte("added <3"))
}

func (handler *BoardHandler) handleMessagesCount(w http.ResponseWriter, r *http.Request) {
	count, err := handler.board.MessagesCount()
	if err != nil {
		log.Errorf("get all messages count error: %s", err)
		w.Write([]byte("error 500: get messages count error"))
		return
	}
	resp := fmt.Sprintf(`{"count":%d}`, count)
	w.Write([]byte(resp))
}

func (handler *BoardHandler) handleGetAllMessages(w http.ResponseWriter, r *http.Request) {
	boardMessages, err := handler.board.AllMessages()
	if err != nil {
		log.Errorf("get all messages error: %s", err)
		w.Write([]byte("error 500: get messages error"))
		return
	}

	messagesJson, err := json.Marshal(boardMessages)
	if err != nil {
		log.Errorf("marshal all messages error: %s", err)
		w.Write([]byte("error 500: get messages error"))
		return
	}

	w.Write(messagesJson)
}
