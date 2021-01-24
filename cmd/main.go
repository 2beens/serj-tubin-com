package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/aerospike"
	as "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("starting ...")

	logLevel := flag.String("loglvl", "trace", "log level")
	forceStart := flag.Bool("force-start", false, "try to force start, regardless of errors")
	aeroHost := flag.String("ahost", "localhost", "hostanme of aerospike server")
	aeroPort := flag.Int("aport", 3000, "aerospike server port number")
	aeroNamespace := flag.String("aero-namespace", "serj-tubin-com", "aerospike namespace value (used in aerospike server)")
	aeroMessagesSet := flag.String("aero-messages-set", "messages", "aerospike set name for board messages (used in aerospike server)")
	port := flag.Int("port", 8080, "port number")
	logsPath := flag.String("logs-path", "", "server logs file path (empty for stdout)")

	aeroSetup := flag.Bool("aero-setup", false, "run aerospike sql setup")
	aeroDataFix := flag.Bool("aero-data-fix", false, "run aerospike sql data fixing / migration")

	flag.Parse()

	if *aeroSetup {
		if err := setupAeroDb(*aeroNamespace, *aeroMessagesSet, *aeroHost, *aeroPort); err != nil {
			fmt.Printf("aero setup failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero setup complete")
		os.Exit(0)
	} else if *aeroDataFix {
		if err := fixAerospikeData(*aeroNamespace, *aeroMessagesSet, *aeroHost, *aeroPort); err != nil {
			fmt.Printf("aero data fix failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero data fix complete")
		os.Exit(0)
	}

	log.Debugf("using port: %d", *port)
	log.Debugf("using server logs path: %s", *logsPath)

	loggingSetup(*logsPath, *logLevel)

	openWeatherApiKey := os.Getenv("OPEN_WEATHER_API_KEY")
	if openWeatherApiKey == "" {
		log.Errorf("open weather API key not set, use OPEN_WEATHER_API_KEY env var to set it")
	}

	versionInfo, err := tryGetLastCommitHash()
	if err != nil {
		log.Tracef("failed to get last commit hash / version info: %s", err)
	} else {
		log.Tracef("running version: %s", versionInfo)
	}

	adminUsername := os.Getenv("SERJ_TUBIN_COM_ADMIN_USERNAME")
	adminPasswordHash := os.Getenv("SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH")
	if adminUsername == "" || adminPasswordHash == "" {
		log.Errorf("admin username and password not set. use SERJ_TUBIN_COM_ADMIN_USERNAME and SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH")
		return
	}

	browserRequestsSecret := os.Getenv("SERJ_BROWSER_REQ_SECRET")
	if browserRequestsSecret == "" {
		log.Errorf("browser secret not set. use SERJ_BROWSER_REQ_SECRET")
	}

	admin := &internal.Admin{
		Username:     adminUsername,
		PasswordHash: adminPasswordHash,
	}

	server, err := internal.NewServer(
		*aeroHost,
		*aeroPort,
		*aeroNamespace,
		*aeroMessagesSet,
		openWeatherApiKey,
		browserRequestsSecret,
		versionInfo,
		admin,
	)
	if err != nil && !*forceStart {
		log.Fatal(err)
	}
	if server != nil {
		server.Serve(*port)
	}
}

// tryGetLastCommitHash will try to get the last commit hash
// assumes that the built main executable is in project root
func tryGetLastCommitHash() (string, error) {
	cmd := exec.Command("/usr/bin/git", "rev-parse", "HEAD")
	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

func loggingSetup(logFileName string, logLevel string) {
	switch strings.ToLower(logLevel) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.TraceLevel)
	}

	if logFileName == "" {
		log.SetOutput(os.Stdout)
		return
	}

	if !strings.HasSuffix(logFileName, ".log") {
		logFileName += ".log"
	}

	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Panicf("failed to open log file %q: %s", logFileName, err)
	}

	log.SetOutput(logFile)
}

func fixAerospikeData(namespace, set, host string, port int) error {
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
	var messages []*internal.BoardMessage
	for rec := range recordSet.Results() {
		if rec.Err != nil {
			return fmt.Errorf("get all messages, record error: %w", rec.Err)
		}

		m := internal.MessageFromBins(aerospike.AeroBinMap(rec.Record.Bins))
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

func setupAeroDb(namespace, set, host string, port int) error {
	fmt.Println("staring aero setup ...")

	aeroClient, err := as.NewClient(host, port)
	if err != nil {
		return fmt.Errorf("failed to create aero client: %w", err)
	}

	// TODO: maybe drop index first ?
	// aeroClient.DropIndex(...)

	if err := createBoardMessagesSecondaryIdIndex(aeroClient, namespace, set); err != nil {
		return err
	}

	// other setup functions when/if needed:

	return nil
}

func createBoardMessagesSecondaryIdIndex(aeroClient *as.Client, namespace, set string) error {
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
