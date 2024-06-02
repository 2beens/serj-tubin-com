package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/2beens/serjtubincom/internal/file_box"
	"github.com/2beens/serjtubincom/internal/logging"

	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("starting file service ...")

	rootPath := flag.String(
		"rootpath",
		"",
		"root path for the files storage",
	)
	env := flag.String("env", "development", "environment [prod | production | dev | development | ddev | dockerdev ]")
	host := flag.String("host", "localhost", "host for the file service")
	port := flag.Int("port", 1987, "port for the file service")
	redisHost := flag.String("rhost", "localhost", "auth service redis host")
	redisPort := flag.Int("rport", 6379, "auth service redis port")

	logToStdout := flag.Bool("log-to-stdout", true, "log to stdout")
	logFilePath := flag.String("log-file-path", "", "path of the log file. empty - not logging to file")
	logLevel := flag.String("log-level", "trace", "log level")
	flag.Parse()

	if *rootPath == "" {
		log.Fatalln("rootpath for files storage not specified")
	}

	redisPassword := os.Getenv("SERJ_REDIS_PASS")
	if redisPassword == "" {
		log.Fatalln("auth service redis password not set. use SERJ_REDIS_PASS")
	}

	if redisPassword == "<skip>" {
		log.Warnln("skipping redis password")
		redisPassword = ""
	}

	if otelServiceName := os.Getenv("OTEL_SERVICE_NAME"); otelServiceName == "" {
		log.Warnln("OTEL_SERVICE_NAME env var not set")
	}

	honeycombEnabled := os.Getenv("HONEYCOMB_ENABLED") == "true"
	if honeycombEnabled {
		if honeycombApiKey := os.Getenv("HONEYCOMB_API_KEY"); honeycombApiKey == "" {
			log.Warnln("HONEYCOMB_API_KEY env var not set")
		}
	} else {
		log.Debugln("honeycomb tracing disabled")
	}

	sentryDSN := os.Getenv("SENTRY_DSN")
	logging.Setup(logging.LoggerSetupParams{
		LogFileName:      *logFilePath,
		LogToStdout:      *logToStdout,
		LogLevel:         *logLevel,
		LogFormatJSON:    false,
		Environment:      *env,
		SentryEnabled:    true,
		SentryDSN:        sentryDSN,
		SentryServerName: "filebox",
	})

	ctx, cancel := context.WithCancel(context.Background())
	chOsInterrupt := make(chan os.Signal, 1)
	signal.Notify(chOsInterrupt, os.Interrupt, syscall.SIGTERM)

	fileService, err := file_box.NewFileService(ctx, *rootPath, *redisHost, *redisPort, redisPassword, honeycombEnabled)
	if err != nil {
		log.Fatalf("failed to create file service: %s", err)
	}

	go fileService.SetupAndServe(*host, *port)

	receivedSig := <-chOsInterrupt
	log.Warnf("signal [%s] received ...", receivedSig)
	cancel()

	// go to sleep 🥱
	fileService.GracefulShutdown()
}
