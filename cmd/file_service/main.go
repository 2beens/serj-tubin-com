package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/logging"
	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("starting file service ...")

	rootPath := flag.String(
		"rootpath",
		"/Users/serj/Documents/projects/serj-tubin-com/test_root",
		"root path for the files storage",
	)
	port := flag.Int("port", 1987, "port for the file service")
	redisHost := flag.String("rhost", "localhost", "auth service redis host")
	redisPort := flag.Int("rport", 6379, "auth service redis port")
	flag.Parse()

	redisPassword := os.Getenv("SERJ_REDIS_PASS")
	if redisPassword == "" {
		log.Fatalln("auth service redis password not set. use SERJ_REDIS_PASS")
	}

	logging.Setup("", true, "debug")

	fileService, err := internal.NewFileService(*rootPath, *redisHost, *redisPort, redisPassword)
	if err != nil {
		log.Fatalf("failed to create file service: %s", err)
	}

	fileService.SetupAndServe("localhost", *port)
}
