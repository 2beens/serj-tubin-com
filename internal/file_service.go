package internal

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/2beens/serjtubincom/internal/file_box"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type FileService struct {
	api file_box.Api
}

func NewFileService(rootPath string) (*FileService, error) {
	api, err := file_box.NewDiskApi(rootPath)
	if err != nil {
		return nil, err
	}
	return &FileService{
		api: api,
	}, nil
}

func (fs *FileService) SetupAndServe(host string, port int) {
	handler := NewFileHandler(fs.api)

	r := mux.NewRouter()
	r.HandleFunc("/folder/root", handler.handleGetRoot).Methods("GET", "OPTIONS")
	r.HandleFunc("/folder/{folderId}/file/{id}", handler.handleGet).Methods("GET", "OPTIONS")
	r.HandleFunc("/folder/{folderId}/files", handler.handleGetFilesList).Methods("GET", "OPTIONS")
	r.HandleFunc("/folder/{folderId}", handler.handleSave).Methods("POST", "OPTIONS")

	ipAndPort := fmt.Sprintf("%s:%d", host, port)

	httpServer := &http.Server{
		Handler:      r,
		Addr:         ipAndPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	chOsInterrupt := make(chan os.Signal, 1)
	signal.Notify(chOsInterrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Infof(" > server listening on: [%s]", ipAndPort)
		log.Fatal(httpServer.ListenAndServe())
	}()

	receivedSig := <-chOsInterrupt

	log.Warnf("signal [%s] received ...", receivedSig)

	// go to sleep 🥱
	fs.gracefulShutdown(httpServer)
}

func (fs *FileService) gracefulShutdown(httpServer *http.Server) {
	maxWaitDuration := time.Second * 10
	ctx, timeoutCancel := context.WithTimeout(context.Background(), maxWaitDuration)
	defer timeoutCancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error(" >>> failed to gracefully shutdown http server")
	}
	log.Warnln("server shut down")
}
