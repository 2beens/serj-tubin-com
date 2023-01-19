package tools

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/2beens/serjtubincom/internal/visitor_board"
	"github.com/2beens/serjtubincom/internal/visitor_board/aerospike"

	as "github.com/aerospike/aerospike-client-go"
)

func SetupAeroDb(namespace, set, host string, port int) error {
	fmt.Println("staring aero setup ...")

	aeroClient, err := as.NewClient(host, port)
	if err != nil {
		return fmt.Errorf("failed to create aero client: %w", err)
	}

	// TODO: maybe drop index first ?
	// aeroClient.DropIndex(...)

	if err := createBoardMessagesSecondaryIndex(aeroClient, namespace, set); err != nil {
		return err
	}

	// other setup functions when/if needed:

	return nil
}

func createBoardMessagesSecondaryIndex(aeroClient *as.Client, namespace, set string) error {
	task, err := aeroClient.CreateIndex(
		nil,
		namespace,
		set,
		"id_index",
		"id",
		as.NUMERIC,
	)
	if err != nil {
		return fmt.Errorf("failed to get create index task: %w", err)
	}

	waitSecondsMax := 20
	for i := 0; i < waitSecondsMax; i++ {
		if done, err := task.IsDone(); err != nil {
			fmt.Println(".")
		} else if done {
			break
		}
		time.Sleep(time.Second)
	}

	if err = <-task.OnComplete(); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func FixAerospikeData(namespace, set, host string, port int) error {
	fmt.Println("staring aero data fix ...")

	aeroClient, err := as.NewClient(host, port)
	if err != nil {
		return fmt.Errorf("failed to create aero client: %w", err)
	}

	recordSet, err := aeroClient.ScanAll(nil, namespace, set)
	if err != nil {
		return fmt.Errorf("failed to scan all messages: %w", err)
	}

	var records []*as.Result
	var messages []*visitor_board.Message
	for rec := range recordSet.Results() {
		if rec.Err != nil {
			return fmt.Errorf("get all messages, record error: %w", rec.Err)
		}

		m := visitor_board.MessageFromBins(aerospike.AeroBinMap(rec.Record.Bins))
		messages = append(messages, &m)
		records = append(records, rec)
	}

	fmt.Printf("received %d messages from aerospike:\n", len(messages))
	for i := range messages {
		msg := messages[i]
		fmt.Printf("%d: %s\n", msg.ID, time.Unix(msg.Timestamp, 0))
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp < messages[j].Timestamp
	})

	fmt.Println()
	fmt.Println("------------------------------------------")
	fmt.Println()

	skipDelete := make(map[int64]bool)
	for i := range messages {
		message := messages[i]
		fmt.Printf("saving message %d: %+v: %s - %s\n", i, time.Unix(message.Timestamp, 0), message.Author, message.Message)

		bins := as.BinMap{
			"id":        i,
			"author":    message.Author,
			"timestamp": message.Timestamp,
			"message":   message.Message,
		}

		key, err := as.NewKey(namespace, set, i)
		if err != nil {
			return fmt.Errorf("failed to create a new message key: %w", err)
		}

		exists, err := aeroClient.Exists(nil, key)
		if err != nil {
			return fmt.Errorf("failed to check message existance of %d: %w", message.Timestamp, err)
		}
		if exists {
			skipDelete[message.Timestamp] = true
			continue
		}

		if err = aeroClient.Put(nil, key, bins); err != nil {
			return fmt.Errorf("failed to save a message [%s] in aero: %w", key, err)
		}
	}

	fmt.Println()
	fmt.Println("------------------------------------------")
	fmt.Println()

	fmt.Println("deleting old records ...")
	for i := range records {
		r := records[i]
		timestamp, ok := r.Record.Bins["timestamp"].(int)
		if !ok {
			fmt.Printf("failed to get timestamp of record %v\n", r)
		}
		if skipDelete[int64(timestamp)] {
			fmt.Printf("skip deleting %d\n", timestamp)
			continue
		}

		fmt.Printf("deleting: %s\n", r.Record.Key)
		deleted, err := aeroClient.Delete(nil, r.Record.Key)
		if err != nil {
			fmt.Printf(" >>> failed to delete %s: %s\n", r.Record.Key, err)
		} else {
			fmt.Printf(" > found and deleted: %t\n", deleted)
		}
	}

	return nil
}

func FixAerospikeMessageIdCounter(namespace, set, host string, port int) error {
	fmt.Printf("staring aero message id counter fix [%s : %s] ...\n", namespace, set)

	aeroClient, err := as.NewClient(host, port)
	if err != nil {
		return fmt.Errorf("failed to create aero client: %w", err)
	}

	metadataSet := set + "-metadata"
	counterExists, err := counterExists(aeroClient, namespace, metadataSet)
	if err != nil {
		fmt.Printf("check counter exists: %s\n", err)
	}
	if counterExists {
		fmt.Printf("messages counter already set, will abort")
		return nil
	}

	spolicy := as.NewScanPolicy()
	spolicy.ConcurrentNodes = true
	spolicy.Priority = as.LOW
	spolicy.IncludeBinData = false

	recs, err := aeroClient.ScanAll(spolicy, namespace, set)
	if err != nil {
		return err
	}

	count := 0
	for range recs.Results() {
		count++
	}

	fmt.Printf("trying to set message id counter to: %d\n", count+1)

	updatedIdCounter, err := setMessageIdCounter(namespace, metadataSet, aeroClient, count+1)
	if err != nil {
		return fmt.Errorf("failed to set message id counter: %w", err)
	}

	fmt.Printf("message id counter set to: %d\n", updatedIdCounter)

	return nil
}

func counterExists(aeroClient *as.Client, namespace, metadataSet string) (bool, error) {
	key, err := as.NewKey(namespace, metadataSet, "message-id-counter")
	if err != nil {
		return false, err
	}

	record, err := aeroClient.Get(nil, key)
	if err != nil {
		return false, err
	}

	counterRaw, ok := record.Bins["idCounter"]
	if !ok {
		return false, errors.New("id counter not existing")
	}

	counter, ok := counterRaw.(int)
	if !ok {
		return false, errors.New("id counter not an integer")
	}

	return counter >= 0, nil
}

func setMessageIdCounter(namespace, set string, aeroClient *as.Client, increment int) (int, error) {
	messageIdCounterKey := "message-id-counter"

	key, err := as.NewKey(namespace, set, messageIdCounterKey)
	if err != nil {
		return -1, err
	}

	counterBin := as.NewBin("idCounter", increment)
	record, err := aeroClient.Operate(nil, key, as.AddOp(counterBin), as.GetOp())
	if err != nil {
		return -1, fmt.Errorf("failed to call aero operate: %w", err)
	}

	counterRaw, ok := record.Bins["idCounter"]
	if !ok {
		log.Printf("\n%+v\n\n", record.Bins)
		return -1, errors.New("id counter not existing")
	}

	counter, ok := counterRaw.(int)
	if !ok {
		return -1, errors.New("id counter not an integer")
	}

	return counter, nil
}
