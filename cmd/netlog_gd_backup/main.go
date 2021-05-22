package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/netlog"
	"gopkg.in/natefinch/lumberjack.v2"
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
	logsPath := flag.String("logs-path", "/var/log/serj-tubin-backend/netlog-backup.log", "server logs file path (empty for stdout)")
	reinit := flag.Bool("reinit", false, "reinitialize all again")
	destroy := flag.Bool("destroy", false, "destroy all files (warning!!) (try running more times, if more than 100 files are present)")

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

	if *destroy {
		if err := netlog.DestroyAllFiles(credentialsFileBytes); err != nil {
			log.Fatalf("destroy failed: %s", err)
		}
		log.Println("destroy done!")
		return
	}

	s, err := netlog.NewGoogleDriveBackupService(credentialsFileBytes)
	if err != nil {
		log.Fatalf("failed to create google drive backup service: %s", err)
	}

	baseTime := time.Now()

	if *reinit {
		if err := s.Reinit(baseTime); err != nil {
			log.Fatalf("reinit failed: %s", err)
		}
		log.Println("reinit done")
		return
	}

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

	log.SetOutput(&lumberjack.Logger{
		Filename:  logFileName,
		MaxSize:   50,    // megabytes
		LocalTime: false, // false -> use UTC
		Compress:  true,  // disabled by default
		// comment out MaxBackups and MaxAge, as I want to retain rotated log files indefinitely for now
		//MaxBackups: 30,
		//MaxAge:     730,   //days
	})
}
