package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/netlog"
)

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
	logsPath := flag.String("logs-path", "", "server logs file path (empty for stdout)")
	reinit := flag.Bool("reinit", false, "reinitialize all again")

	flag.Parse()

	loggingSetup(*logsPath)

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
	credentialsFileBytes, err := ioutil.ReadFile(*credentialsFile)
	if err != nil {
		log.Fatalf("unable to read client secret file: %v", err)
	}

	s, err := netlog.NewGoogleDriveBackupService(credentialsFileBytes)
	if err != nil {
		log.Fatalf("failed to create google drive backup service: %s", err)
	}

	if *reinit {
		if err := s.Reinit(); err != nil {
			log.Fatalf("reinit failed: %s", err)
		}
		log.Println("reinit done, will backup now ...")
	}

	baseTime := time.Now()
	if err := s.DoBackup(baseTime); err != nil {
		log.Fatalf("%+v", err)
	}
}

func loggingSetup(logFileName string) {
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
