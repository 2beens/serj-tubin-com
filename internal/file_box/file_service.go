package file_box

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type FileService struct {
	api          *DiskApi
	loginChecker *auth.LoginChecker
}

func NewFileService(
	rootPath string,
	redisHost string,
	redisPort int,
	redisPassword string,
) (*FileService, error) {
	api, err := NewDiskApi(rootPath)
	if err != nil {
		return nil, err
	}

	authServiceRedisEndpoint := net.JoinHostPort(redisHost, fmt.Sprintf("%d", redisPort))
	log.Debugf("connecting to auth service redis: %s", authServiceRedisEndpoint)

	rdb := redis.NewClient(&redis.Options{
		Addr:     authServiceRedisEndpoint,
		Password: redisPassword,
		DB:       0, // use default DB
	})

	rdbStatus := rdb.Ping(context.Background())
	if err := rdbStatus.Err(); err != nil {
		log.Errorf("--> failed to ping redis: %s", err)
	} else {
		log.Printf("redis ping: %s", rdbStatus.Val())
	}

	return &FileService{
		api:          api,
		loginChecker: auth.NewLoginChecker(auth.DefaultTTL, rdb),
	}, nil
}

func RouterSetup(handler *FileHandler) *mux.Router {
	r := mux.NewRouter()

	fileServiceRouter := r.PathPrefix("/f").Subrouter()
	fileServiceRouter.HandleFunc("/root", handler.handleGetRoot).Methods("GET", "OPTIONS")
	fileServiceRouter.HandleFunc("/{folderId}/c/{id}", handler.handleUpdateInfo).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/del/{folderId}/c/{id}", handler.handleDelete).Methods("DELETE", "OPTIONS")
	fileServiceRouter.HandleFunc("/del/{folderId}", handler.handleDeleteFolder).Methods("DELETE", "OPTIONS")
	fileServiceRouter.HandleFunc("/upload/{folderId}", handler.handleUpload).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/{parentId}/new", handler.handleNewFolder).Methods("POST", "OPTIONS")
	// get a file content
	r.HandleFunc("/link/{folderId}/c/{id}", handler.handleGet).Methods("GET", "OPTIONS")

	r.Use(middleware.LogRequest())
	r.Use(middleware.Cors())
	fileServiceRouter.Use(handler.authMiddleware())

	r.Use(middleware.DrainAndCloseRequest())

	return r
}

func (fs *FileService) SetupAndServe(host string, port int) {
	handler := NewFileHandler(fs.api, fs.loginChecker)
	r := RouterSetup(handler)

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
