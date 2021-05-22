package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/logging"
	"github.com/2beens/serjtubincom/tools"
	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("starting ...")

	logLevel := flag.String("loglvl", "trace", "log level")
	forceStart := flag.Bool("force-start", false, "try to force start, regardless of errors")
	aeroHost := flag.String("ahost", "localhost", "hostname of aerospike server")
	aeroPort := flag.Int("aport", 3000, "aerospike server port number")
	aeroNamespace := flag.String("aero-namespace", "serj-tubin-com", "aerospike namespace value (used in aerospike server)")
	aeroMessagesSet := flag.String("aero-messages-set", "messages", "aerospike set name for board messages (used in aerospike server)")
	port := flag.Int("port", 8080, "port number")
	logToStdout := flag.Bool("o", true, "additionally, write logs to stdout")
	logsPath := flag.String("logs-path", "/var/log/serj-tubin-backend/service.log", "server logs file path (empty for stdout)")

	aeroSetup := flag.Bool("aero-setup", false, "run aerospike sql setup")
	aeroDataFix := flag.Bool("aero-data-fix", false, "run aerospike sql data fixing / migration")
	aeroMessageIdCounterSet := flag.Bool("aero-msg-id-counter-set", false, "set / fix the id counter for visitor board messages")

	flag.Parse()

	if *aeroSetup {
		if err := tools.SetupAeroDb(*aeroNamespace, *aeroMessagesSet, *aeroHost, *aeroPort); err != nil {
			fmt.Printf("aero setup failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero setup completed")
		os.Exit(0)
	} else if *aeroDataFix {
		if err := tools.FixAerospikeData(*aeroNamespace, *aeroMessagesSet, *aeroHost, *aeroPort); err != nil {
			fmt.Printf("aero data fix failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero data fix completed")
		os.Exit(0)
	} else if *aeroMessageIdCounterSet {
		if err := tools.FixAerospikeMessageIdCounter(*aeroNamespace, *aeroMessagesSet, *aeroHost, *aeroPort); err != nil {
			fmt.Printf("aero message id counter fix / set failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("\naero message id counter fix / set completed")
		os.Exit(0)
	}

	log.Debugf("using port: %d", *port)
	log.Debugf("using server logs path: %s", *logsPath)

	logging.Setup(*logsPath, *logToStdout, *logLevel)

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
