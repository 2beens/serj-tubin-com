package file_box

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type FileService struct {
	api           *DiskApi
	loginChecker  *auth.LoginChecker
	httpServer    *http.Server
	shutdownFuncs []func()
}

func NewFileService(
	ctx context.Context,
	rootPath string,
	redisHost string,
	redisPort int,
	redisPassword string,
	honeycombTracingEnabled bool,
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
		log.Debugf("redis ping: %s", rdbStatus.Val())
	}

	// use honeycomb distro to setup OpenTelemetry SDK
	otelShutdown, err := tracing.HoneycombSetup(honeycombTracingEnabled, "file-service", rdb)
	if err != nil {
		return nil, err
	}

	shutdownFuncs := []func(){
		otelShutdown,
		func() {
			if err := rdb.Close(); err != nil {
				log.Errorf("redis close: %s", err)
			}
		},
	}

	return &FileService{
		api:           api,
		loginChecker:  auth.NewLoginChecker(auth.DefaultLoginSessionTTL, rdb),
		shutdownFuncs: shutdownFuncs,
	}, nil
}

func RouterSetup(handler *FileHandler, metricsManager *metrics.Manager) *mux.Router {
	r := mux.NewRouter()

	fileServiceRouter := r.PathPrefix("/f").Subrouter()
	fileServiceRouter.HandleFunc("/root", handler.handleGetRoot).Methods("GET", "OPTIONS")
	fileServiceRouter.HandleFunc("/update/{id}", handler.handleUpdateInfo).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/del", handler.handleDelete).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/upload/{folderId}", handler.handleUpload).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/{parentId}/new", handler.handleNewFolder).Methods("POST", "OPTIONS")
	fileServiceRouter.HandleFunc("/download/folder/{folderId}", handler.handleDownloadFolder).Methods("GET", "OPTIONS")
	fileServiceRouter.HandleFunc("/download/file/{id}", handler.handleDownloadFile).Methods("GET", "OPTIONS")

	fileServiceRouter.Use(handler.authMiddleware())

	// get a file content
	r.HandleFunc("/link/{id}", handler.handleGet).Methods("GET", "OPTIONS")

	r.Use(middleware.PanicRecovery(metricsManager))
	r.Use(middleware.LogRequest())
	r.Use(middleware.RequestMetrics(metricsManager))
	r.Use(middleware.Cors())
	r.Use(middleware.DrainAndCloseRequest())

	return r
}

func (fs *FileService) SetupAndServe(host string, port int) {
	promRegistry := metrics.SetupPrometheus()
	metricsManager := metrics.NewManager("backend", "filebox", promRegistry)

	handler := NewFileHandler(fs.api, fs.loginChecker)
	r := RouterSetup(handler, metricsManager)

	ipAndPort := fmt.Sprintf("%s:%d", host, port)
	fs.httpServer = &http.Server{
		Handler:      otelhttp.NewHandler(r, "file-service"),
		Addr:         ipAndPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Infof(" > server listening on: [%s]", ipAndPort)
	err := fs.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("file service, listen and serve: %s", err)
	}
}

func (fs *FileService) GracefulShutdown() {
	log.Debugln("shutting down ...")

	for _, sf := range fs.shutdownFuncs {
		sf()
	}

	if fs.httpServer != nil {
		maxWaitDuration := time.Second * 10
		ctx, timeoutCancel := context.WithTimeout(context.Background(), maxWaitDuration)
		defer timeoutCancel()

		log.Debugln("shutting down server ...")
		if err := fs.httpServer.Shutdown(ctx); err != nil {
			log.Error(" >>> failed to gracefully shutdown http server")
		}
	}

	log.Warnln("server shut down")
}
