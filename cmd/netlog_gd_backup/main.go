package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/db"
	"github.com/2beens/serjtubincom/internal/logging"
	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	log "github.com/sirupsen/logrus"
)

// netlog google drive backup cmd

func main() {
	credentialsFile := flag.String(
		"gd-creds",
		"./lazar-dusan-veliki-drive-credentials.json",
		"lazar dusan google drive credentials json",
	)
	tokenFile := flag.String(
		"token-file",
		"./token.json",
		"google drive token file json",
	)
	logToStdout := flag.Bool("o", true, "additionally, write logs to stdout")
	logsPath := flag.String("logs-path", "/var/log/serj-tubin-backend/netlog-backup.log", "server logs file path (empty for stdout)")
	reinit := flag.Bool("reinit", false, "reinitialize all again")
	destroy := flag.Bool("destroy", false, "destroy all files (warning!!) (try running more times, if more than 100 files are present)")
	env := flag.String("env", "development", "environment [prod | production | dev | development | ddev | dockerdev]")
	configPath := flag.String("config", "./config.toml", "path for the TOML config file")
	flag.Parse()

	cfg, err := config.Load(*env, *configPath)
	if err != nil {
		panic(err)
	}

	sentryDSN := os.Getenv("SENTRY_DSN")
	logging.Setup(logging.LoggerSetupParams{
		LogFileName:      *logsPath,
		LogToStdout:      *logToStdout,
		LogLevel:         "trace",
		LogFormatJSON:    false,
		Environment:      cfg.Environment,
		SentryEnabled:    cfg.SentryEnabled,
		SentryDSN:        sentryDSN,
		SentryServerName: "netlog-gd-backup",
	})

	log.Println("staring netlog backup ...")

	if *credentialsFile == "" {
		log.Fatalln("google drive credentials json not specified")
	}
	if *tokenFile == "" {
		log.Fatalln("google drive token file json not specified")
	}
	if *reinit {
		log.Println("!! attention: will reinitialize all again...")
	}

	// lazar.dusan.veliki@gmail.com // stara sifra
	credentialsFileBytes, err := os.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("unable to read client secret file: %v", err)
	}

	chOsInterrupt := make(chan os.Signal, 1)
	signal.Notify(chOsInterrupt, os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		receivedSig := <-chOsInterrupt
		log.Warnf("signal [%s] received, canceling context ...", receivedSig)
		cancel()
	}()

	if *destroy {
		if err := netlog.DestroyAllFiles(ctx, credentialsFileBytes); err != nil {
			log.Fatalf("destroy failed: %s", err)
		}
		log.Println("destroy done!")
		return
	}

	// TODO: enable tracing here

	honeycombEnabled := os.Getenv("HONEYCOMB_ENABLED") == "true"
	if honeycombEnabled {
		honeycombConfig := tracing.ReadHoneycombConfig()
		if err := tracing.ValidateHoneycombConfig(honeycombConfig); err != nil {
			log.Fatalf("honeycomb config invalid: %s", err)
		}
	} else {
		log.Debugln("honeycomb tracing disabled")
	}

	dbPool, err := db.NewDBPool(ctx, db.NewDBPoolParams{
		DBHost:         cfg.PostgresHost,
		DBPort:         cfg.PostgresPort,
		DBName:         cfg.PostgresDBName,
		TracingEnabled: honeycombEnabled,
	})
	if err != nil {
		log.Fatalf("new db pool: %s", err)
	}
	defer dbPool.Close()

	s, err := netlog.NewGoogleDriveBackupService(
		ctx,
		credentialsFileBytes,
		dbPool,
		cfg.NetlogUnixSocketAddrDir,
		cfg.NetlogUnixSocketFileName,
	)
	if err != nil {
		log.Fatalf("failed to create google drive backup service: %s", err)
	}

	baseTime := time.Now()

	if *reinit {
		if err := s.Reinit(ctx, baseTime); err != nil {
			log.Fatalf("reinit failed: %s", err)
		}
		log.Println("reinit done")
		return
	}

	if err := s.DoBackup(ctx, baseTime); err != nil {
		log.Fatalf("%+v", err)
	}
}
