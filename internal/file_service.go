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
	"github.com/2beens/serjtubincom/internal/middleware"
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
	r.HandleFunc("/f/root", handler.handleGetRoot).Methods("GET", "OPTIONS")
	r.HandleFunc("/f/{folderId}/c/{id}", handler.handleGet).Methods("GET", "OPTIONS")
	r.HandleFunc("/f/{folderId}/c/{id}", handler.handleDelete).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/f/{folderId}", handler.handleDeleteFolder).Methods("DELETE", "OPTIONS")
	r.HandleFunc("/f/{folderId}", handler.handleSave).Methods("POST", "OPTIONS")
	r.HandleFunc("/f/{parentId}/new", handler.handleNewFolder).Methods("POST", "OPTIONS")
	r.HandleFunc("/f/{folderId}/c", handler.handleGetFilesList).Methods("GET", "OPTIONS")

	r.Use(middleware.LogRequest())
	r.Use(middleware.Cors())
	r.Use(middleware.DrainAndCloseRequest())

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
