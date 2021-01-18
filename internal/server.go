package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/2beens/serjtubincom/internal/cache"
	"github.com/2beens/serjtubincom/internal/netlog"
	as "github.com/aerospike/aerospike-client-go"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	OneHour            = 60 * 60
	GeoIpCacheExpire   = OneHour * 5 // default expire in hours
	WeatherCacheExpire = OneHour * 1 // default expire in hours
)

type Server struct {
	blogApi         blog.BlogApi
	geoIp           *GeoIp
	quotesManager   *QuotesManager
	board           *Board
	netlogVisitsApi *netlog.VisitApi

	browserRequestsSecret string // used in netlog, when posting new visit

	openWeatherAPIUrl string
	openWeatherApiKey string
	muteRequestLogs   bool
	versionInfo       string

	loginSession *LoginSession
	admin        *Admin
}

func NewServer(
	aerospikeHost string,
	aerospikePort int,
	aeroNamespace string,
	aeroMessagesSet string,
	openWeatherApiKey string,
	browserRequestsSecret string,
	versionInfo string,
	admin *Admin,
) (*Server, error) {
	log.Debugf("connecting to aerospike server %s:%d [namespace:%s, set:%s] ...",
		aerospikeHost, aerospikePort, aeroNamespace, aeroMessagesSet)

	aeroClient, err := as.NewClient(aerospikeHost, aerospikePort)
	if err != nil {
		return nil, fmt.Errorf("failed to create aero client: %w", err)
	}
	boardAeroClient, err := aerospike.NewBoardAeroClient(aeroClient, aeroNamespace, aeroMessagesSet)
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

	blogApi, err := blog.NewBlogPsqlApi()
	if err != nil {
		log.Fatalf("failed to create blog api: %s", err)
	}

	netlogVisitsApi, err := netlog.NewVisitApi()
	if err != nil {
		log.Fatalf("failed to create netlog visits api: %s", err)
	}

	s := &Server{
		blogApi:               blogApi,
		openWeatherAPIUrl:     "http://api.openweathermap.org/data/2.5",
		openWeatherApiKey:     openWeatherApiKey,
		browserRequestsSecret: browserRequestsSecret,
		muteRequestLogs:       false,
		geoIp:                 NewGeoIp("https://freegeoip.app", http.DefaultClient),
		board:                 board,
		netlogVisitsApi:       netlogVisitsApi,
		versionInfo:           versionInfo,
		loginSession:          &LoginSession{},
		admin:                 admin,
	}

	qm, err := NewQuoteManager("./assets/quotes.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to create quote manager: %s", err)
	}

	s.quotesManager = qm

	return s, nil
}

func (s *Server) routerSetup() (*mux.Router, error) {
	r := mux.NewRouter()

	blogRouter := r.PathPrefix("/blog").Subrouter()
	weatherRouter := r.PathPrefix("/weather").Subrouter()
	boardRouter := r.PathPrefix("/board").Subrouter()
	netlogRouter := r.PathPrefix("/netlog").Subrouter()

	if NewBlogHandler(blogRouter, s.blogApi, s.loginSession) == nil {
		return nil, errors.New("blog handler is nil")
	}

	if NewBoardHandler(boardRouter, s.board, s.loginSession) == nil {
		return nil, errors.New("board handler is nil")
	}

	if weatherHandler, err := NewWeatherHandler(weatherRouter, s.geoIp, s.openWeatherAPIUrl, s.openWeatherApiKey); err != nil {
		return nil, fmt.Errorf("failed to create weather handler: %w", err)
	} else if weatherHandler == nil {
		return nil, errors.New("weather handler is nil")
	}

	if NewMiscHandler(r, s.geoIp, s.quotesManager, s.versionInfo, s.loginSession, s.admin) == nil {
		panic("misc handler is nil")
	}

	if NewNetlogHandler(netlogRouter, s.netlogVisitsApi, s.browserRequestsSecret, s.loginSession) == nil {
		panic("netlog visits handler is nil")
	}

	r.Use(s.loggingMiddleware())
	r.Use(s.corsMiddleware())

	return r, nil
}

func (s *Server) Serve(port int) {
	router, err := s.routerSetup()
	if err != nil {
		log.Fatalf("failed to setup router: %s", err)
	}

	ipAndPort := fmt.Sprintf("%s:%d", "localhost", port)

	httpServer := &http.Server{
		Handler:      router,
		Addr:         ipAndPort,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	chOsInterrupt := make(chan os.Signal, 1)
	signal.Notify(chOsInterrupt, os.Interrupt)

	go func() {
		log.Infof(" > server listening on: [%s]", ipAndPort)
		log.Fatal(httpServer.ListenAndServe())
	}()

	for {
		<-chOsInterrupt
		log.Warn("os interrupt received ...")
		s.gracefulShutdown(httpServer)
	}
}

func (s *Server) gracefulShutdown(httpServer *http.Server) {
	log.Debug("graceful shutdown initiated ...")

	s.board.Close()

	if s.blogApi != nil {
		s.blogApi.CloseDB()
	}

	maxWaitDuration := time.Second * 15
	ctx, cancel := context.WithTimeout(context.Background(), maxWaitDuration)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error(" >>> failed to gracefully shutdown http server")
	}
	log.Warn("server shut down")
}

func (s *Server) corsMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//Allow CORS here By * or specific origin
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) loggingMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.muteRequestLogs {
				userAgent := r.Header.Get("User-Agent")
				log.Tracef(" ====> request [%s] path: [%s] [UA: %s]", r.Method, r.URL.Path, userAgent)
			}
			next.ServeHTTP(w, r)
		})
	}
}
