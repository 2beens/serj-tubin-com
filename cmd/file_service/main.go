package main

import (
	"fmt"

	"github.com/2beens/serjtubincom/internal"
	"github.com/2beens/serjtubincom/internal/logging"
	log "github.com/sirupsen/logrus"
)

func main() {
	fmt.Println("starting file service ...")

	logging.Setup("", true, "debug")

	// TODO: for config
	host := "localhost"
	port := 1987
	rootPath := "/Users/serj/Projects/serj-tubin-com/test_root"

	fileService, err := internal.NewFileService(rootPath)
	if err != nil {
		log.Fatalf("failed to create file service: %s", err)
	}

	fileService.SetupAndServe(host, port)
}
