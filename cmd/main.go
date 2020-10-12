package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/2beens/serjtubincom/internal"
)

func main() {
	fmt.Println("starting ...")

	forceStart := flag.Bool("force-start", false, "try to force start, regardless of errors")
	aeroHost := flag.String("ahost", "localhost", "hostanme of aerospike server")
	aeroPort := flag.Int("aport", 3000, "aerospike server port number")
	aeroNamespace := flag.String("aero-namespace", "serj-tubin-com", "aerospike namespace value (used in aerospike server)")
	port := flag.Int("port", 8080, "port number")
	logsPath := flag.String("logs-path", "", "server logs file path (empty for stdout)")
	flag.Parse()
	log.Debugf("using port: %d", *port)
	log.Debugf("using server logs path: %s", *logsPath)

	loggingSetup(*logsPath, "trace")

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
