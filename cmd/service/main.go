package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/logging"
	"github.com/2beens/serjtubincom/tools"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("starting ...")

	env := flag.String("env", "development", "environment [prod | production | dev | development]")
	configPath := flag.String("config", "./config.toml", "path for the TOML config file")
	// TODO: extract to separate cmd, it's super ugly
	aeroSetup := flag.Bool("aero-setup", false, "run aerospike sql setup")
	aeroDataFix := flag.Bool("aero-data-fix", false, "run aerospike sql data fixing / migration")
	aeroMessageIdCounterSet := flag.Bool("aero-msg-id-counter-set", false, "set / fix the id counter for visitor board messages")

	flag.Parse()

	var tomlConfig config.Toml
	if _, err := toml.DecodeFile(*configPath, &tomlConfig); err != nil {
		panic(err)
	}

	cfg, err := tomlConfig.Get(*env)
	if err != nil {
		panic(err)
	}

	if *aeroSetup {
		if err := tools.SetupAeroDb(cfg.AeroNamespace, cfg.AeroMessagesSet, cfg.AeroHost, cfg.AeroPort); err != nil {
			fmt.Printf("aero setup failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero setup completed")
		os.Exit(0)
	} else if *aeroDataFix {
		if err := tools.FixAerospikeData(cfg.AeroNamespace, cfg.AeroMessagesSet, cfg.AeroHost, cfg.AeroPort); err != nil {
			fmt.Printf("aero data fix failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero data fix completed")
		os.Exit(0)
	} else if *aeroMessageIdCounterSet {
		if err := tools.FixAerospikeMessageIdCounter(cfg.AeroNamespace, cfg.AeroMessagesSet, cfg.AeroHost, cfg.AeroPort); err != nil {
			fmt.Printf("aero message id counter fix / set failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero message id counter fix / set completed")
		os.Exit(0)
	}

	log.Debugf("using port: %d", cfg.Port)
	log.Debugf("using server logs path: %s", cfg.LogsPath)

	logging.Setup(cfg.LogsPath, cfg.LogToStdout, cfg.LogLevel)

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
		cfg.AeroHost,
		cfg.AeroPort,
		cfg.AeroNamespace,
		cfg.AeroMessagesSet,
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
	return string(stdout), nil
}
