package internal

import (
	"fmt"
	"strconv"

	as "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

// TODO: unit tests <3

type BoardMessage struct {
	Author    string `json:"author"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

type Board struct {
	aeroClient     *as.Client
	boardNamespace string
	messagesSet    string
}

func NewBoard(aeroHost string, aeroPort int, namespace string) (*Board, error) {
	log.Debugf("connecting to aerospike server %s:%d ...", aeroHost, aeroPort)

	client, err := as.NewClient(aeroHost, aeroPort)
	if err != nil {
		return nil, err
	}

	b := &Board{
		aeroClient:     client,
		boardNamespace: namespace,
		messagesSet:    "messages",
	}

	return b, nil
}

// TODO: implement server graceful shutdown
func (b *Board) Close() {
	if b.aeroClient != nil {
		b.aeroClient.Close()
	}
}

func (b *Board) StoreMessage(message BoardMessage) error {
	if b.aeroClient == nil {
		return fmt.Errorf("aero client is nil")
	}

	key, err := as.NewKey(b.boardNamespace, b.messagesSet, message.Timestamp)
	if err != nil {
		return err
	}

	log.Debugf("saving message: %+v: %s - %s", message.Timestamp, message.Author, message.Message)

	bins := as.BinMap{
		"author":    message.Author,
		"timestamp": message.Timestamp,
		"message":   message.Message,
	}

	writePolicy := as.NewWritePolicy(0, 0)
	err = b.aeroClient.Put(writePolicy, key, bins)
	if err != nil {
		return err
	}

	return nil
}

func (b *Board) AllMessages() ([]*BoardMessage, error) {
	if b.aeroClient == nil {
		return nil, fmt.Errorf("aero client is nil")
	}

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = true

	recs, err := b.aeroClient.ScanAll(spolicy, b.boardNamespace, b.messagesSet)
	if err != nil {
		return nil, err
	}

	// TODO: maybe try getting all keys first, and initiate a batch Read ?

	var messages []*BoardMessage
	for rec := range recs.Results() {
		if rec.Err != nil {
			log.Errorf("get all messages, record error: %s", err)
			continue
		}

		log.Println("BINS: %+v", rec.Record.Bins)
		t := rec.Record.Bins["timestamp"]
		log.Printf("%d of type %T", t, t)

		author, ok := rec.Record.Bins["author"].(string)
		if !ok {
			log.Errorf("get all messages, convert author to string failed!")
		}
		timestampStr, ok := rec.Record.Bins["timestamp"].(string)
		if !ok {
			log.Errorf("1 get all messages, convert timestamp (%+v) to int failed!", timestampStr)
		}
		message, ok := rec.Record.Bins["message"].(string)
		if !ok {
			log.Errorf("get all messages, convert message to string failed!")
		}

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err == nil {
			log.Errorf("2 get all messages, convert timestamp (%+v) to int failed!", timestampStr)
		}

		messages = append(messages, &BoardMessage{
			Author:    author,
			Timestamp: timestamp,
			Message:   message,
		})
	}

	return messages, nil
}

func (b *Board) MessagesCount() (int, error) {
	if b.aeroClient == nil {
		return -1, fmt.Errorf("aero client is nil")
	}

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = false

	recs, err := b.aeroClient.ScanAll(spolicy, b.boardNamespace, b.messagesSet)
	if err != nil {
		return -1, err
	}

	count := 0
	for _ = range recs.Results() {
		count++
	}

	return count, nil
}
