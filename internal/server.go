package internal

import (
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/db"
	"github.com/2beens/serjtubincom/internal/geoip"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/misc"
	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/2beens/serjtubincom/internal/notes_box"
	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	metricsmiddleware "github.com/2beens/serjtubincom/internal/telemetry/metrics/middleware"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	visitorBoard "github.com/2beens/serjtubincom/internal/visitor_board"
	"github.com/2beens/serjtubincom/internal/weather"

	"github.com/IBM/pgxpoolprometheus"
	"github.com/getsentry/sentry-go"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Server struct {
	httpServer *http.Server

	config          *config.Config
	dbPool          *pgxpool.Pool
	blogApi         *blog.PsqlApi
	geoIp           *geoip.Api
	weatherApi      *weather.Api
	quotesManager   *misc.QuotesManager
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

type NewServerParams struct {
	Config                  *config.Config
	OpenWeatherApiKey       string
	IpInfoAPIKey            string
	BrowserRequestsSecret   string
	VersionInfo             string
	AdminUsername           string
	AdminPasswordHash       string
	RedisPassword           string
	HoneycombTracingEnabled bool
}

func NewServer(
	ctx context.Context,
	params NewServerParams,
) (*Server, error) {
	dbPool, err := db.NewDBPool(ctx, db.NewDBPoolParams{
		DBHost:         params.Config.PostgresHost,
		DBPort:         params.Config.PostgresPort,
		DBName:         params.Config.PostgresDBName,
		TracingEnabled: params.HoneycombTracingEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("new db pool: %w", err)
	}

	blogApi, err := blog.NewBlogPsqlApi(
		ctx,
		params.Config.PostgresHost, params.Config.PostgresPort, params.Config.PostgresDBName,
		true,
	)
	if err != nil {
		log.Fatalf("failed to create blog api: %s", err)
	}

	netlogVisitsApi, err := netlog.NewNetlogPsqlApi(
		ctx,
		params.Config.PostgresHost, params.Config.PostgresPort, params.Config.PostgresDBName,
		true,
	)
	if err != nil {
		log.Fatalf("failed to create netlog visits api: %s", err)
	}

	notesBoxApi, err := notes_box.NewPsqlApi(dbPool)
	if err != nil {
		log.Fatalf("failed to create notes visits api: %s", err)
	}

	pgxpoolCollector := pgxpoolprometheus.NewCollector(
		dbPool,
		map[string]string{"db_name": "serj_tubin_com_db"},
	)
	promRegistry := metrics.SetupPrometheus(pgxpoolCollector)
	metricsManager := metrics.NewManager("backend", "main", promRegistry)
	metricsManager.GaugeLifeSignal.Set(0) // will be set to 1 when all is set and ran (I think this is probably not needed)

	rdb := redis.NewClient(&redis.Options{
		Addr:     net.JoinHostPort(params.Config.RedisHost, params.Config.RedisPort),
		Password: params.RedisPassword,
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
	otelShutdown, err := tracing.HoneycombSetup(params.HoneycombTracingEnabled, "main-backend", rdb)
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
		config:                params.Config,
		dbPool:                dbPool,
		blogApi:               blogApi,
		browserRequestsSecret: params.BrowserRequestsSecret,
		geoIp:                 geoip.NewApi(geoip.DefaultIpInfoBaseURL, params.IpInfoAPIKey, tracedHttpClient, rdb),
		weatherApi: weather.NewApi(
			"http://api.openweathermap.org/data/2.5",
			params.OpenWeatherApiKey,
			weatherCitiesData,
			tracedHttpClient,
		),
		netlogVisitsApi: netlogVisitsApi,
		notesBoxApi:     notesBoxApi,
		versionInfo:     params.VersionInfo,

		redisClient:  rdb,
		authService:  authService,
		loginChecker: auth.NewLoginChecker(auth.DefaultTTL, rdb),
		admin: &auth.Admin{
			Username:     params.AdminUsername,
			PasswordHash: params.AdminPasswordHash,
		},

		// telemetry
		metricsManager: metricsManager,
		promRegistry:   promRegistry,
		otelShutdown:   otelShutdown,
	}

	quotesCsvFile, err := os.Open(params.Config.QuotesCsvPath)
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

func (s *Server) routerSetup(ctx context.Context) (*mux.Router, error) {
	r := mux.NewRouter()
	r.Use(otelmux.Middleware("main-router"))

	blogHandler := blog.NewBlogHandler(s.blogApi, s.loginChecker)
	blogHandler.SetupRoutes(r)

	boardRepo := visitorBoard.NewRepo(s.dbPool)
	boardHandler := visitorBoard.NewBoardHandler(boardRepo, s.loginChecker)
	boardHandler.SetupRoutes(r)

	weatherHandler := weather.NewHandler(s.geoIp, s.weatherApi)
	r.HandleFunc("/weather/current", weatherHandler.HandleCurrent).Methods("GET")
	r.HandleFunc("/weather/tomorrow", weatherHandler.HandleTomorrow).Methods("GET")
	r.HandleFunc("/weather/5days", weatherHandler.Handle5Days).Methods("GET")

	reqRateLimiter := redis_rate.NewLimiter(s.redisClient)
	miscHandler := misc.NewHandler(s.geoIp, s.quotesManager, s.versionInfo, s.authService, s.admin)
	miscHandler.SetupRoutes(r, reqRateLimiter, s.metricsManager)

	netlogHandler := netlog.NewHandler(s.netlogVisitsApi, s.metricsManager, s.browserRequestsSecret, s.loginChecker)
	netlogHandler.SetupRoutes(r)

	notesHandler := notes_box.NewHandler(s.notesBoxApi, s.loginChecker, s.metricsManager)
	r.HandleFunc("/notes", notesHandler.HandleList).Methods("GET", "OPTIONS").Name("list-notes")
	r.HandleFunc("/notes", notesHandler.HandleAdd).Methods("POST", "OPTIONS").Name("new-note")
	r.HandleFunc("/notes", notesHandler.HandleUpdate).Methods("PUT", "OPTIONS").Name("update-note")
	r.HandleFunc("/notes/{id}", notesHandler.HandleDelete).Methods("DELETE", "OPTIONS").Name("remove-note")

	// all the rest - unhandled paths
	r.HandleFunc("/{unknown}", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}).Methods("GET", "POST", "PUT", "OPTIONS").Name("unknown")

	authMiddleware := middleware.NewAuthMiddlewareHandler(
		s.browserRequestsSecret,
		s.loginChecker,
	)

	r.Use(middleware.PanicRecovery(s.metricsManager))
	r.Use(middleware.LogRequest())
	r.Use(middleware.RequestMetrics(s.metricsManager))
	r.Use(middleware.Cors())
	r.Use(authMiddleware.AuthCheck())
	r.Use(middleware.DrainAndCloseRequest())

	return r, nil
}

func (s *Server) Serve(ctx context.Context, host string, port int) {
	router, err := s.routerSetup(ctx)
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

	if s.dbPool != nil {
		log.Traceln("closing db pool ...")
		s.dbPool.Close() // blocking operation
		log.Trace("db pool closed")
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

	if ok := sentry.Flush(5 * time.Second); ok {
		log.Debugf("sentry flush ok: %t", ok)
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

func (s *Server) setNetlogBackupUnixSocket(ctx context.Context) {
	if err := os.MkdirAll(s.config.NetlogUnixSocketAddrDir, os.ModePerm); err != nil {
		log.Errorf("failed to create netlog backup unix socket dir: %s", err)
		return
	}

	if addr, err := netlog.VisitsBackupUnixSocketListenerSetup(
		ctx,
		s.config.NetlogUnixSocketAddrDir,
		s.config.NetlogUnixSocketFileName,
		s.metricsManager,
	); err != nil {
		log.Errorf("failed to create netlog backup unix socket: %s", err)
	} else {
		log.Debugf("netlog backup unix socket: %s", addr)
	}
}
