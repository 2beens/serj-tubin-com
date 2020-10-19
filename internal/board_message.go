package internal

import (
	"github.com/2beens/serjtubincom/internal/aerospike"
	log "github.com/sirupsen/logrus"
)

type BoardMessage struct {
	ID        int    `json:"id"`
	Author    string `json:"author"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

// TODO: maybe better return error on fail ot get any of the fields
func MessageFromBins(bins aerospike.AeroBinMap) BoardMessage {
	id, ok := bins["id"].(int)
	if !ok {
		log.Errorln("get all messages, convert id to int failed!")
	}
	author, ok := bins["author"].(string)
	if !ok {
		log.Errorln("get all messages, convert author to string failed!")
	}
	message, ok := bins["message"].(string)
	if !ok {
		log.Errorln("get all messages, convert message to string failed!")
	}

	boardMessage := BoardMessage{
		ID:      id,
		Author:  author,
		Message: message,
	}

	if timestamp, ok := bins["timestamp"].(int); ok {
		boardMessage.Timestamp = int64(timestamp)
	} else if timestamp, ok := bins["timestamp"].(int64); ok {
		boardMessage.Timestamp = timestamp
	} else {
		log.Errorln("get all messages, convert timestamp to int/int64 failed!")
	}

	return boardMessage
}
