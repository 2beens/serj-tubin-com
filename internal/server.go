package internal

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/geoip"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/misc"
	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/2beens/serjtubincom/internal/notes_box"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	metricsmiddleware "github.com/2beens/serjtubincom/internal/telemetry/metrics/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/internal/visitor_board"
	"github.com/2beens/serjtubincom/internal/visitor_board/aerospike"
	"github.com/2beens/serjtubincom/internal/weather"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Server struct {
	httpServer *http.Server

	config          *config.Config
	blogApi         *blog.PsqlApi
	geoIp           *geoip.Api
	weatherApi      *weather.Api
	quotesManager   *misc.QuotesManager
	boardAeroClient *aerospike.BoardAeroClient
	boardClient     *visitor_board.Client
	netlogVisitsApi *netlog.PsqlApi
	notesBoxApi     *notes_box.PsqlApi

	browserRequestsSecret string // used in netlog, when posting new visit

	versionInfo string

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
		return nil, fmt.Errorf("failed to create visitor_board aero client: %w", err)
	}

	boardCache, err := visitor_board.NewBoardCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create visitor_board cache: %w", err)
	}

	boardClient, err := visitor_board.NewClient(ctx, boardAeroClient, boardCache)
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

	rdbStatus := rdb.Ping(context.Background())
	if err := rdbStatus.Err(); err != nil {
		log.Errorf("--> failed to ping redis: %s", err)
	} else {
		log.Printf("redis ping: %s", rdbStatus.Val())
	}

	authService := auth.NewAuthService(auth.DefaultTTL, rdb)
	go func() {
		for range time.Tick(time.Hour * 8) {
			authService.ScanAndClean(ctx)
		}
	}()

	// use honeycomb distro to setup OpenTelemetry SDK
	otelShutdown, err := tracing.HoneycombSetup(honeycombTracingEnabled, "main-backend", rdb)
	if err != nil {
		return nil, err
	}

	tracedHttpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	weatherCitiesData, err := weather.LoadCitiesData()
	if err != nil {
		log.Errorf("failed to load weather cities data: %s", err)
		return nil, fmt.Errorf("failed to load weather cities data: %s", err)
	}

	s := &Server{
		config:                config,
		blogApi:               blogApi,
		browserRequestsSecret: browserRequestsSecret,
		geoIp:                 geoip.NewApi(geoip.DefaultIpInfoBaseURL, ipInfoAPIKey, tracedHttpClient, rdb),
		weatherApi: weather.NewApi(
			"http://api.openweathermap.org/data/2.5",
			openWeatherApiKey,
			weatherCitiesData,
			tracedHttpClient,
		),
		boardAeroClient: boardAeroClient,
		boardClient:     boardClient,
		netlogVisitsApi: netlogVisitsApi,
		notesBoxApi:     notesBoxApi,
		versionInfo:     versionInfo,

		redisClient:  rdb,
		authService:  authService,
		loginChecker: auth.NewLoginChecker(auth.DefaultTTL, rdb),
		admin: &auth.Admin{
			Username:     adminUsername,
			PasswordHash: adminPasswordHash,
		},

		// telemetry
		metricsManager: metricsManager,
		promRegistry:   promRegistry,
		otelShutdown:   otelShutdown,
	}

	quotesCsvFile, err := os.Open(config.QuotesCsvPath)
	if err != nil {
		return nil, fmt.Errorf("open quotes file: %w", err)
	}
	defer func() {
		if err := quotesCsvFile.Close(); err != nil {
			log.Warnf("close quotes csv file: %s", err)
		}
	}()

	s.quotesManager, err = misc.NewQuoteManager(csv.NewReader(quotesCsvFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create quote manager: %s", err)
	}

	return s, nil
}

func (s *Server) routerSetup() (*mux.Router, error) {
	r := mux.NewRouter()
	r.Use(otelmux.Middleware("main-router"))

	blogRouter := r.PathPrefix("/blog").Subrouter()
	weatherRouter := r.PathPrefix("/weather").Subrouter()
	boardRouter := r.PathPrefix("/board").Subrouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	notesRouter := r.PathPrefix("/notes").Subrouter()

	blogHandler := blog.NewBlogHandler(s.blogApi, s.loginChecker)
	blogHandler.SetupRoutes(blogRouter)

	boardHandler := visitor_board.NewBoardHandler(s.boardClient, s.loginChecker)
	boardHandler.SetupRoutes(boardRouter)

	weatherHandler := weather.NewHandler(s.geoIp, s.weatherApi)
	weatherRouter.HandleFunc("/current", weatherHandler.HandleCurrent).Methods("GET")
	weatherRouter.HandleFunc("/tomorrow", weatherHandler.HandleTomorrow).Methods("GET")
	weatherRouter.HandleFunc("/5days", weatherHandler.Handle5Days).Methods("GET")

	reqRateLimiter := redis_rate.NewLimiter(s.redisClient)
	miscHandler := misc.NewHandler(s.geoIp, s.quotesManager, s.versionInfo, s.authService, s.admin)
	miscHandler.SetupRoutes(r, reqRateLimiter, s.metricsManager)

	netlogHandler := netlog.NewHandler(s.netlogVisitsApi, s.metricsManager, s.browserRequestsSecret, s.loginChecker)
	netlogHandler.SetupRoutes(netlogRouter)

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

	ipAndPort := net.JoinHostPort(host, strconv.Itoa(port))
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

	go func() {
		metricsAddr := net.JoinHostPort(s.config.PrometheusMetricsHost, s.config.PrometheusMetricsPort)
		log.Printf(" > metrics listening on: [%s]", metricsAddr)

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

	// TODO: check if prometheus data has to be flushed before total shutdown

	s.otelShutdown()
	log.Trace("otel shut down ...")

	if s.redisClient != nil {
		if err := s.redisClient.Close(); err != nil {
			log.Errorf("failed to close redis client conn: %s", err)
		}
	}

	if s.boardAeroClient != nil {
		s.boardAeroClient.Close()
		log.Trace("visitor_board aero client closed")
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
