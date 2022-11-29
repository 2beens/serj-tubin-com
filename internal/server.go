package internal

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/2beens/serjtubincom/internal/board"
	"github.com/2beens/serjtubincom/internal/board/aerospike"
	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/geoip"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/misc"
	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/2beens/serjtubincom/internal/notes_box"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	metricsmiddleware "github.com/2beens/serjtubincom/internal/telemetry/metrics/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/internal/weather"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

type Server struct {
	httpServer *http.Server

	config          *config.Config
	blogApi         *blog.PsqlApi
	geoIp           *geoip.Api
	quotesManager   *misc.QuotesManager
	boardAeroClient *aerospike.BoardAeroClient
	boardClient     *board.Client
	netlogVisitsApi *netlog.PsqlApi
	notesBoxApi     *notes_box.PsqlApi

	browserRequestsSecret string // used in netlog, when posting new visit

	openWeatherAPIUrl string
	openWeatherApiKey string
	versionInfo       string

	redisClient  *redis.Client
	loginChecker *auth.LoginChecker
	authService  *auth.Service
	admin        *auth.Admin

	// metrics
	metricsManager *metrics.Manager
	promRegistry   *prometheus.Registry
	otelShutdown   func()
}

func NewServer(
	ctx context.Context,
	config *config.Config,
	openWeatherApiKey string,
	ipInfoAPIKey string,
	browserRequestsSecret string,
	versionInfo string,
	adminUsername string,
	adminPasswordHash string,
	redisPassword string,
	honeycombTracingEnabled bool,
) (*Server, error) {
	boardAeroClient, err := aerospike.NewBoardAeroClient(config.AeroHost, config.AeroPort, config.AeroNamespace, config.AeroMessagesSet)
	if err != nil {
		return nil, fmt.Errorf("failed to create board aero client: %w", err)
	}

	boardCache, err := board.NewBoardCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create board cache: %w", err)
	}

	boardClient, err := board.NewClient(ctx, boardAeroClient, boardCache)
	if err != nil {
		return nil, fmt.Errorf("failed to create visitor board: %s", err)
	}

	if openWeatherApiKey == "" {
		log.Errorf("error getting Weather info: open weather api key not set")
		return nil, errors.New("open weather API key not set")
	}

	blogApi, err := blog.NewBlogPsqlApi(
		ctx,
		config.PostgresHost, config.PostgresPort, config.PostgresDBName,
		true,
	)
	if err != nil {
		log.Fatalf("failed to create blog api: %s", err)
	}

	netlogVisitsApi, err := netlog.NewNetlogPsqlApi(
		ctx,
		config.PostgresHost, config.PostgresPort, config.PostgresDBName,
		true,
	)
	if err != nil {
		log.Fatalf("failed to create netlog visits api: %s", err)
	}

	notesBoxApi, err := notes_box.NewPsqlApi(
		ctx,
		config.PostgresHost, config.PostgresPort, config.PostgresDBName,
		true,
	)
	if err != nil {
		log.Fatalf("failed to create notes visits api: %s", err)
	}

	promRegistry := metrics.SetupPrometheus()
	metricsManager := metrics.NewManager("backend", "main", promRegistry)
	metricsManager.GaugeLifeSignal.Set(0) // will be set to 1 when all is set and ran (I think this is probably not needed)

	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(config.RedisHost, config.RedisPort),
		Password: redisPassword,
		DB:       0, // use default DB
	})

	// tracing support for redis client
	rdb.AddHook(redisotel.NewTracingHook(
		redisotel.WithAttributes(attribute.String("component", "main-backend"))),
	)

	rdbStatus := rdb.Ping(context.Background())
	if err := rdbStatus.Err(); err != nil {
		log.Errorf("--> failed to ping redis: %s", err)
	} else {
		log.Printf("redis ping: %s", rdbStatus.Val())
	}

	authService := auth.NewAuthService(auth.DefaultTTL, rdb)
	// if config.IsDev {
	// 	authService.RandStringFunc = func(s int) (string, error) {
	// 		return "test-token", nil
	// 	}
	// 	if t, err := authService.Login(time.Now()); err != nil || t != "test-token" {
	// 		panic("test auth service failed to initialize")
	// 	}
	// }

	loginChecker := auth.NewLoginChecker(auth.DefaultTTL, rdb)

	go func() {
		for range time.Tick(time.Hour * 8) {
			authService.ScanAndClean(ctx)
		}
	}()

	admin := &auth.Admin{
		Username:     adminUsername,
		PasswordHash: adminPasswordHash,
	}

	// use honeycomb distro to setup OpenTelemetry SDK
	otelShutdown, err := tracing.HoneycombSetup(honeycombTracingEnabled)
	if err != nil {
		return nil, err
	}

	tracedHttpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	s := &Server{
		config:                config,
		blogApi:               blogApi,
		openWeatherAPIUrl:     "http://api.openweathermap.org/data/2.5",
		openWeatherApiKey:     openWeatherApiKey,
		browserRequestsSecret: browserRequestsSecret,
		geoIp:                 geoip.NewApi(ipInfoAPIKey, tracedHttpClient, rdb),
		boardAeroClient:       boardAeroClient,
		boardClient:           boardClient,
		netlogVisitsApi:       netlogVisitsApi,
		notesBoxApi:           notesBoxApi,
		versionInfo:           versionInfo,

		redisClient:  rdb,
		authService:  authService,
		loginChecker: loginChecker,
		admin:        admin,

		//metrics
		metricsManager: metricsManager,
		promRegistry:   promRegistry,
		otelShutdown:   otelShutdown,
	}

	quotesCsvFile, err := os.Open(config.QuotesCsvPath)
	if err != nil {
		return nil, fmt.Errorf("open quotes file: %w", err)
	}
	defer quotesCsvFile.Close()

	s.quotesManager, err = misc.NewQuoteManager(csv.NewReader(quotesCsvFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create quote manager: %s", err)
	}

	return s, nil
}

func (s *Server) routerSetup() (*mux.Router, error) {
	r := mux.NewRouter()

	// TODO: it should do some degree of auto tracing, but it does not
	// update: actually, seems like adds some tracing info
	r.Use(otelmux.Middleware("main-router"))

	blogRouter := r.PathPrefix("/blog").Subrouter()
	weatherRouter := r.PathPrefix("/weather").Subrouter()
	boardRouter := r.PathPrefix("/board").Subrouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	notesRouter := r.PathPrefix("/notes").Subrouter()

	// TODO: refactor this - return handlers, but define routes here, similar to notes handler
	if blog.NewBlogHandler(blogRouter, s.blogApi, s.loginChecker) == nil {
		return nil, errors.New("blog handler is nil")
	}

	if board.NewBoardHandler(boardRouter, s.boardClient, s.loginChecker) == nil {
		return nil, errors.New("board handler is nil")
	}

	if weatherHandler, err := weather.NewHandler(weatherRouter, s.geoIp, s.openWeatherAPIUrl, s.openWeatherApiKey); err != nil {
		return nil, fmt.Errorf("failed to create weather handler: %w", err)
	} else if weatherHandler == nil {
		return nil, errors.New("weather handler is nil")
	}

	if misc.NewHandler(r, s.geoIp, s.quotesManager, s.versionInfo, s.authService, s.admin) == nil {
		panic("misc handler is nil")
	}

	if netlog.NewHandler(netlogRouter, s.netlogVisitsApi, s.metricsManager, s.browserRequestsSecret, s.loginChecker) == nil {
		panic("netlog visits handler is nil")
	}

	notesHandler := notes_box.NewHandler(s.notesBoxApi, s.loginChecker, s.metricsManager)
	notesRouter.HandleFunc("", notesHandler.HandleList).Methods("GET", "OPTIONS").Name("list-notes")
	notesRouter.HandleFunc("", notesHandler.HandleAdd).Methods("POST", "OPTIONS").Name("new-note")
	notesRouter.HandleFunc("", notesHandler.HandleUpdate).Methods("PUT", "OPTIONS").Name("update-note")
	notesRouter.HandleFunc("/{id}", notesHandler.HandleDelete).Methods("DELETE", "OPTIONS").Name("remove-note")
	notesRouter.Use(notesHandler.AuthMiddleware())

	// all the rest - unhandled paths
	r.HandleFunc("/{unknown}", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}).Methods("GET", "POST", "PUT", "OPTIONS").Name("unknown")

	r.Use(middleware.PanicRecovery(s.metricsManager))
	r.Use(middleware.RequestMetrics(s.metricsManager))
	r.Use(middleware.LogRequest())
	r.Use(middleware.Cors())
	r.Use(middleware.DrainAndCloseRequest())

	return r, nil
}

func (s *Server) Serve(ctx context.Context, host string, port int) {
	router, err := s.routerSetup()
	if err != nil {
		log.Fatalf("failed to setup router: %s", err)
	}

	ipAndPort := fmt.Sprintf("%s:%d", host, port)

	s.httpServer = &http.Server{
		Handler:      router,
		Addr:         ipAndPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		ConnState:    s.connStateMetrics,
	}

	go func() {
		log.Infof(" > server listening on: [%s]", ipAndPort)
		log.Fatal(s.httpServer.ListenAndServe())
	}()

	metricsAddr := net.JoinHostPort(s.config.PrometheusMetricsHost, s.config.PrometheusMetricsPort)
	log.Printf(" > metrics listening on: [%s]", metricsAddr)
	go func() {
		// Expose the registered metrics via HTTP.
		http.Handle(
			"/metrics",
			metricsmiddleware.
				New(s.promRegistry, nil).
				WrapHandler("/metrics", promhttp.HandlerFor(
					s.promRegistry,
					promhttp.HandlerOpts{}),
				))
		log.Println(http.ListenAndServe(metricsAddr, nil))
	}()

	s.metricsManager.GaugeLifeSignal.Set(1)

	// netlog backup unix socket
	s.setNetlogBackupUnixSocket(ctx)
}

func (s *Server) setNetlogBackupUnixSocket(ctx context.Context) {
	if err := os.MkdirAll(s.config.NetlogUnixSocketAddrDir, os.ModePerm); err != nil {
		log.Errorf("failed to create netlog backup unix socket dir: %s", err)
		return
	}

	if addr, err := netlog.VisitsBackupUnixSocketListenerSetup(ctx, s.config.NetlogUnixSocketAddrDir, s.config.NetlogUnixSocketFileName, s.metricsManager); err != nil {
		log.Errorf("failed to create netlog backup unix socket: %s", err)
	} else {
		log.Debugf("netlog backup unix socket: %s", addr)
	}
}

func (s *Server) GracefulShutdown() {
	log.Debug("graceful shutdown initiated ...")

	// TODO: probably not needed to be set explicitly
	s.metricsManager.GaugeLifeSignal.Set(0)

	s.otelShutdown()
	log.Trace("otel shut down ...")

	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			log.Errorf("failed to close redis client conn: %s", err)
		}
	}

	if s.boardAeroClient != nil {
		s.boardAeroClient.Close()
		log.Trace("board aero client closed")
	}
	if s.netlogVisitsApi != nil {
		s.netlogVisitsApi.CloseDB()
		log.Trace("netlog visits api closed")
	}
	if s.blogApi != nil {
		s.blogApi.CloseDB()
		log.Trace("blog api closed")
	}

	log.Debugln("removing netlog backup unix socket ...")
	if err := os.RemoveAll(s.config.NetlogUnixSocketAddrDir); err != nil {
		log.Errorf("failed to cleanup netlog backup unix socket dir: %s", err)
	}

	maxWaitDuration := time.Second * 15
	ctx, timeoutCancel := context.WithTimeout(context.Background(), maxWaitDuration)
	defer timeoutCancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Error(" >>> failed to gracefully shutdown http server")
	}
	log.Warnln("server shut down")
}

func (s *Server) connStateMetrics(_ net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		s.metricsManager.GaugeRequests.Add(1)
	case http.StateClosed:
		s.metricsManager.GaugeRequests.Add(-1)
	}
}
