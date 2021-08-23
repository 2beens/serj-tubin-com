package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

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
		cfg,
		openWeatherApiKey,
		browserRequestsSecret,
		versionInfo,
		admin,
	)
	if err != nil {
		log.Fatal(err)
	}

	server.Serve(cfg.Port)
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
