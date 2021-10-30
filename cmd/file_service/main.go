package main

import (
	"fmt"
	"os"

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
	rootPath := "/Users/serj/Documents/projects/serj-tubin-com/test_root"

	adminUsername := os.Getenv("SERJ_TUBIN_COM_ADMIN_USERNAME")
	adminPasswordHash := os.Getenv("SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH")
	if adminUsername == "" || adminPasswordHash == "" {
		log.Errorf("admin username and password not set. use SERJ_TUBIN_COM_ADMIN_USERNAME and SERJ_TUBIN_COM_ADMIN_PASSWORD_HASH")
		return
	}

	admin := &internal.Admin{
		Username:     adminUsername,
		PasswordHash: adminPasswordHash,
	}

	fileService, err := internal.NewFileService(rootPath, admin)
	if err != nil {
		log.Fatalf("failed to create file service: %s", err)
	}

	fileService.SetupAndServe(host, port)
}
