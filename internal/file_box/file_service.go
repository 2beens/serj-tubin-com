package file_box

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
	httpServer   *http.Server
}

func NewFileService(
	ctx context.Context,
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

	rdbStatus := rdb.Ping(ctx)
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
	fileServiceRouter.HandleFunc("/update/{id}", handler.handleUpdateInfo).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/del", handler.handleDelete).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/upload/{folderId}", handler.handleUpload).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/{parentId}/new", handler.handleNewFolder).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/download/folder/{folderId}", handler.handleDownloadFolder).Methods("GET", "OPTIONS")
	fileServiceRouter.HandleFunc("/download/file/{id}", handler.handleDownloadFile).Methods("GET", "OPTIONS")
	// get a file content
	r.HandleFunc("/link/{id}", handler.handleGet).Methods("GET", "OPTIONS")

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
	fs.httpServer = &http.Server{
		Handler:      r,
		Addr:         ipAndPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof(" > server listening on: [%s]", ipAndPort)
	log.Fatal(fs.httpServer.ListenAndServe())
}

func (fs *FileService) GracefulShutdown() {
	if fs.httpServer != nil {
		maxWaitDuration := time.Second * 10
		ctx, timeoutCancel := context.WithTimeout(context.Background(), maxWaitDuration)
		defer timeoutCancel()
		if err := fs.httpServer.Shutdown(ctx); err != nil {
			log.Error(" >>> failed to gracefully shutdown http server")
		}
	}
	log.Warnln("server shut down")
}
