package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/logging"
	"github.com/2beens/serjtubincom/pkg"
	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("starting ...")

	env := flag.String("env", "development", "environment [prod | production | dev | development | ddev | doockerdev ]")
	configPath := flag.String("config", "./config.toml", "path for the TOML config file")
	flag.Parse()

	log.Warnf("---->> running in [%s] environment", *env)

	cfg, err := config.Load(*env, *configPath)
	if err != nil {
		panic(err)
	}

	logging.Setup(cfg.LogsPath, cfg.LogToStdout, cfg.LogLevel)

	log.Debugf("using port: %d", cfg.Port)
	log.Debugf("using server logs path: [%s]", cfg.LogsPath)

	openWeatherApiKey := os.Getenv("OPEN_WEATHER_API_KEY")
	if openWeatherApiKey == "" {
		log.Errorf("open weather API key not set, use OPEN_WEATHER_API_KEY env var to set it")
	}

	ipBaseAPIKey := os.Getenv("IP_BASE_API_KEY")
	if ipBaseAPIKey == "" {
		log.Errorf("ip base API key not set, use IP_BASE_API_KEY env var to set it")
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

	redisPassword := os.Getenv("SERJ_REDIS_PASS")
	if redisPassword == "" {
		log.Errorf("redis password not set. use SERJ_REDIS_PASS")
	}

	chOsInterrupt := make(chan os.Signal, 1)
	signal.Notify(chOsInterrupt, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())

	server, err := internal.NewServer(
		ctx,
		cfg,
		openWeatherApiKey,
		ipBaseAPIKey,
		browserRequestsSecret,
		versionInfo,
		adminUsername,
		adminPasswordHash,
		redisPassword,
	)
	if err != nil {
		log.Fatal(err)
	}

	server.Serve(ctx, cfg.Host, cfg.Port)

	receivedSig := <-chOsInterrupt
	log.Warnf("signal [%s] received, killing everything ...", receivedSig)
	cancel()

	// go to sleep 🥱
	server.GracefulShutdown()
}

// tryGetLastCommitHash will try to get the last commit hash
// assumes that the built main executable is in project root
func tryGetLastCommitHash() (string, error) {
	cmd := exec.Command("/usr/bin/git", "rev-parse", "HEAD")
	stdout, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return pkg.BytesToString(stdout), nil
}
