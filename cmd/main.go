package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal"
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
	port := flag.Int("port", 8080, "port number")
	logsPath := flag.String("logs-path", "", "server logs file path (empty for stdout)")
	aeroSetup := flag.Bool("aero-setup", false, "run aerospike db setup")

	flag.Parse()

	if *aeroSetup {
		if err := setupAeroDb(*aeroNamespace, *aeroHost, *aeroPort); err != nil {
			fmt.Printf("aero setup failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("aero setup complete")
		os.Exit(0)
	}

	log.Debugf("using port: %d", *port)
	log.Debugf("using server logs path: %s", *logsPath)

	loggingSetup(*logsPath, *logLevel)

	openWeatherApiKey := os.Getenv("OPEN_WEATHER_API_KEY")
	if openWeatherApiKey == "" {
		log.Errorf("open weather API key not set, use OPEN_WEATHER_API_KEY env var to set it")
	}

	server, err := internal.NewServer(*aeroHost, *aeroPort, *aeroNamespace, openWeatherApiKey)
	if err != nil && !*forceStart {
		log.Fatal(err)
	}
	if server != nil {
		server.Serve(*port)
	}
}

func setupAeroDb(namespace, host string, port int) error {
	fmt.Println("staring aero setup ...")

	aeroClient, err := as.NewClient(host, port)
	if err != nil {
		return fmt.Errorf("failed to create aero client: %w", err)
	}

	// TODO: maybe drop index first ?
	// aeroClient.DropIndex(...)

	if err := createBoardMessagesSecondaryIndex(aeroClient, namespace); err != nil {
		return err
	}

	return nil
}

func createBoardMessagesSecondaryIndex(aeroClient *as.Client, namespace string) error {
	task, err := aeroClient.CreateIndex(
		nil,
		namespace,
		"messages",
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
