package internal

import (
	as "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

type BoardMessage struct {
	Author    string `json:"author"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

func MessageFromBins(bins as.BinMap) BoardMessage {
	author, ok := bins["author"].(string)
	if !ok {
		log.Errorln("get all messages, convert author to string failed!")
	}
	timestamp, ok := bins["timestamp"].(int)
	if !ok {
		log.Errorln("get all messages, convert timestamp to int failed!")
	}
	message, ok := bins["message"].(string)
	if !ok {
		log.Errorln("get all messages, convert message to string failed!")
	}
	return BoardMessage{
		Author:    author,
		Timestamp: int64(timestamp),
		Message:   message,
	}
}
