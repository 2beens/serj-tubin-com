package internal

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/2beens/serjtubincom/internal/cache"
	"github.com/2beens/serjtubincom/internal/config"
	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/2beens/serjtubincom/internal/middleware"
	"github.com/2beens/serjtubincom/internal/netlog"
	"github.com/2beens/serjtubincom/internal/notes_box"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const (
	OneHour            = 60 * 60
	GeoIpCacheExpire   = OneHour * 5 // default expire in hours
	WeatherCacheExpire = OneHour * 1 // default expire in hours
)

type Server struct {
	config          *config.Config
	redisClient     *redis.Client
	blogApi         blog.Api
	geoIp           *GeoIp
	quotesManager   *QuotesManager
	board           *Board
	netlogVisitsApi *netlog.PsqlApi
	notesBoxApi     *notes_box.PsqlApi

	browserRequestsSecret string // used in netlog, when posting new visit

	openWeatherAPIUrl string
	openWeatherApiKey string
	versionInfo       string

	authService *auth.Service
	admin       *auth.Admin

	// metrics
	instr *instrumentation.Instrumentation
}

func NewServer(
	config *config.Config,
	openWeatherApiKey string,
	browserRequestsSecret string,
	versionInfo string,
	adminUsername string,
	adminPasswordHash string,
) (*Server, error) {
	boardAeroClient, err := aerospike.NewBoardAeroClient(config.AeroHost, config.AeroPort, config.AeroNamespace, config.AeroMessagesSet)
	if err != nil {
		return nil, fmt.Errorf("failed to create board aero client: %w", err)
	}

	boardCache, err := cache.NewBoardCache()
	if err != nil {
		return nil, fmt.Errorf("failed to create board cache: %w", err)
	}

	board, err := NewBoard(boardAeroClient, boardCache)
	if err != nil {
		return nil, fmt.Errorf("failed to create visitor board: %s", err)
	}

	if openWeatherApiKey == "" {
		log.Errorf("error getting Weather info: open weather api key not set")
		return nil, errors.New("open weather API key not set")
	}

	blogApi, err := blog.NewBlogPsqlApi(config.PostgresHost, config.PostgresPort, config.PostgresDBName)
	if err != nil {
		log.Fatalf("failed to create blog api: %s", err)
	}

	netlogVisitsApi, err := netlog.NewNetlogPsqlApi(config.PostgresHost, config.PostgresPort, config.PostgresDBName)
	if err != nil {
		log.Fatalf("failed to create netlog visits api: %s", err)
	}

	notesBoxApi, err := notes_box.NewPsqlApi(config.PostgresHost, config.PostgresPort, config.PostgresDBName)
	if err != nil {
		log.Fatalf("failed to create notes visits api: %s", err)
	}

	instr := instrumentation.NewInstrumentation("backend", "server1")
	instr.GaugeLifeSignal.Set(0) // will be set to 1 when all is set and ran (I think this is probably not needed)

	rdb := redis.NewClient(&redis.Options{
		Addr: net.JoinHostPort(config.RedisHost, config.RedisPort),
		// TODO:
		// Password: redisPass,
		DB: 0, // use default DB
	})

	rdbStatus := rdb.Ping(context.Background())
	if err := rdbStatus.Err(); err != nil {
		log.Errorf("--> failed to ping redis: %s", err)
	} else {
		log.Printf("redis ping: %s", rdbStatus.Val())
	}

	// TODO: make configurable ?
	ttl := 24 * 7 * time.Hour // max login session duration - 7 days
	authService := auth.NewAuthService(ttl, rdb)
	// if config.IsDev {
	// 	devLoginSession := &LoginSession{
	// 		Token:     "test-token",
	// 		CreatedAt: time.Now(),
	// 	}
	// 	authService.sessions[devLoginSession.Token] = devLoginSession
	// }

	go func() {
		for range time.Tick(time.Hour * 8) {
			authService.ScanAndClean()
		}
	}()

	admin := &auth.Admin{
		Username:     adminUsername,
		PasswordHash: adminPasswordHash,
	}

	s := &Server{
		config:                config,
		blogApi:               blogApi,
		openWeatherAPIUrl:     "http://api.openweathermap.org/data/2.5",
		openWeatherApiKey:     openWeatherApiKey,
		browserRequestsSecret: browserRequestsSecret,
		geoIp:                 NewGeoIp("https://freegeoip.app", http.DefaultClient),
		board:                 board,
		netlogVisitsApi:       netlogVisitsApi,
		notesBoxApi:           notesBoxApi,
		versionInfo:           versionInfo,
		authService:           authService,
		admin:                 admin,

		//metrics
		instr: instr,
	}

	s.quotesManager, err = NewQuoteManager("./assets/quotes.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to create quote manager: %s", err)
	}

	return s, nil
}

func (s *Server) routerSetup() (*mux.Router, error) {
	r := mux.NewRouter()

	blogRouter := r.PathPrefix("/blog").Subrouter()
	weatherRouter := r.PathPrefix("/weather").Subrouter()
	boardRouter := r.PathPrefix("/board").Subrouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()
	notesRouter := r.PathPrefix("/notes").Subrouter()

	// TODO: refactor this - return handlers, but define routes here, similar to notes handler
	if NewBlogHandler(blogRouter, s.blogApi, s.authService) == nil {
		return nil, errors.New("blog handler is nil")
	}

	if NewBoardHandler(boardRouter, s.board, s.authService) == nil {
		return nil, errors.New("board handler is nil")
	}

	if weatherHandler, err := NewWeatherHandler(weatherRouter, s.geoIp, s.openWeatherAPIUrl, s.openWeatherApiKey); err != nil {
		return nil, fmt.Errorf("failed to create weather handler: %w", err)
	} else if weatherHandler == nil {
		return nil, errors.New("weather handler is nil")
	}

	if NewMiscHandler(r, s.geoIp, s.quotesManager, s.versionInfo, s.authService, s.admin) == nil {
		panic("misc handler is nil")
	}

	if NewNetlogHandler(netlogRouter, s.netlogVisitsApi, s.instr, s.browserRequestsSecret, s.authService) == nil {
		panic("netlog visits handler is nil")
	}

	notesHandler := NewNotesBoxHandler(s.notesBoxApi, s.authService, s.instr)
	notesRouter.HandleFunc("", notesHandler.handleList).Methods("GET", "OPTIONS").Name("list-notes")
	notesRouter.HandleFunc("", notesHandler.handleAdd).Methods("POST", "OPTIONS").Name("new-note")
	notesRouter.HandleFunc("", notesHandler.handleUpdate).Methods("PUT", "OPTIONS").Name("update-note")
	notesRouter.HandleFunc("/{id}", notesHandler.handleDelete).Methods("DELETE", "OPTIONS").Name("remove-note")
	notesRouter.Use(notesHandler.authMiddleware())

	// all the rest - unhandled paths
	r.HandleFunc("/{unknown}", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}).Methods("GET", "POST", "PUT", "OPTIONS").Name("unknown")

	r.Use(middleware.PanicRecovery(s.instr))
	r.Use(middleware.LogRequest())
	r.Use(middleware.Cors())
	r.Use(middleware.DrainAndCloseRequest())
	r.Use(middleware.RequestMetrics(s.instr))

	return r, nil
}

func (s *Server) Serve(host string, port int) {
	router, err := s.routerSetup()
	if err != nil {
		log.Fatalf("failed to setup router: %s", err)
	}

	ipAndPort := fmt.Sprintf("%s:%d", host, port)

	httpServer := &http.Server{
		Handler:      router,
		Addr:         ipAndPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		ConnState:    s.connStateMetrics,
	}

	chOsInterrupt := make(chan os.Signal, 1)
	signal.Notify(chOsInterrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Infof(" > server listening on: [%s]", ipAndPort)
		log.Fatal(httpServer.ListenAndServe())
	}()

	metricsAddr := net.JoinHostPort(s.config.PrometheusMetricsHost, s.config.PrometheusMetricsPort)
	log.Printf(" > metrics listening on: [%s]", metricsAddr)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println(http.ListenAndServe(metricsAddr, nil))
	}()

	// netlog backup unix socket
	ctx, cancel := context.WithCancel(context.Background())
	s.setNetlogBackupUnixSocket(ctx)

	s.instr.GaugeLifeSignal.Set(1)
	receivedSig := <-chOsInterrupt

	log.Warnf("signal [%s] received ...", receivedSig)
	s.instr.GaugeLifeSignal.Set(0)

	// go to sleep 🥱
	s.gracefulShutdown(httpServer, cancel)
}

func (s *Server) setNetlogBackupUnixSocket(ctx context.Context) {
	if err := os.MkdirAll(s.config.NetlogUnixSocketAddrDir, os.ModePerm); err != nil {
		log.Errorf("failed to create netlog backup unix socket dir: %s", err)
		return
	}

	if addr, err := netlog.VisitsBackupUnixSocketListenerSetup(ctx, s.config.NetlogUnixSocketAddrDir, s.config.NetlogUnixSocketFileName, s.instr); err != nil {
		log.Errorf("failed to create netlog backup unix socket: %s", err)
	} else {
		log.Debugf("netlog backup unix socket: %s", addr)
	}
}

func (s *Server) gracefulShutdown(httpServer *http.Server, cancel context.CancelFunc) {
	log.Debug("graceful shutdown initiated ...")

	cancel()

	s.board.Close()

	if s.blogApi != nil {
		s.blogApi.CloseDB()
	}

	log.Debugln("removing netlog backup unix socket ...")
	if err := os.RemoveAll(s.config.NetlogUnixSocketAddrDir); err != nil {
		log.Errorf("failed to cleanup netlog backup unix socket dir: %s", err)
	}

	maxWaitDuration := time.Second * 15
	ctx, timeoutCancel := context.WithTimeout(context.Background(), maxWaitDuration)
	defer timeoutCancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error(" >>> failed to gracefully shutdown http server")
	}
	log.Warn("server shut down")
}

func (s *Server) connStateMetrics(_ net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		s.instr.GaugeRequests.Add(1)
	case http.StateClosed:
		s.instr.GaugeRequests.Add(-1)
	}
}
