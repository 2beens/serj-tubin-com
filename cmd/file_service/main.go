package main

import (
	"flag"
	"fmt"
	"os"

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
	port := flag.Int("port", 1987, "port for the file service")
	redisHost := flag.String("rhost", "localhost", "auth service redis host")
	redisPort := flag.Int("rport", 6379, "auth service redis port")
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

	logging.Setup("", true, "debug")

	fileService, err := file_box.NewFileService(*rootPath, *redisHost, *redisPort, redisPassword)
	if err != nil {
		log.Fatalf("failed to create file service: %s", err)
	}

	fileService.SetupAndServe("localhost", *port)
}
