package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/aerospike"
	"github.com/2beens/serjtubincom/internal/blog"
	"github.com/2beens/serjtubincom/internal/cache"
	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/2beens/serjtubincom/internal/netlog"
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
	blogApi         blog.Api
	geoIp           *GeoIp
	quotesManager   *QuotesManager
	board           *Board
	netlogVisitsApi *netlog.PsqlApi

	browserRequestsSecret string // used in netlog, when posting new visit

	openWeatherAPIUrl string
	openWeatherApiKey string
	muteRequestLogs   bool
	versionInfo       string

	loginSession *LoginSession
	admin        *Admin

	// metrics
	instr *instrumentation.Instrumentation
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
	boardAeroClient, err := aerospike.NewBoardAeroClient(aerospikeHost, aerospikePort, aeroNamespace, aeroMessagesSet)
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

	netlogVisitsApi, err := netlog.NewNetlogPsqlApi()
	if err != nil {
		log.Fatalf("failed to create netlog visits api: %s", err)
	}

	instrumentation := instrumentation.NewInstrumentation("backend", "server1")
	instrumentation.GaugeLifeSignal.Set(0) // will be set to 1 when all is set and ran

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

		//metrics
		instr: instrumentation,
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

	if NewNetlogHandler(netlogRouter, s.netlogVisitsApi, s.instr, s.browserRequestsSecret, s.loginSession) == nil {
		panic("netlog visits handler is nil")
	}

	r.Use(s.panicRecoveryMiddleware())
	r.Use(s.loggingMiddleware())
	r.Use(s.corsMiddleware())
	r.Use(s.drainAndCloseMiddleware())
	r.Use(s.metricsMiddleware())

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
		ConnState:    s.connStateMetrics,
	}

	chOsInterrupt := make(chan os.Signal, 1)
	signal.Notify(chOsInterrupt, os.Interrupt)

	go func() {
		log.Infof(" > server listening on: [%s]", ipAndPort)
		log.Fatal(httpServer.ListenAndServe())
	}()

	// TODO: make metrics settings configurable
	metricsPort := "2112"
	metricsAddr := net.JoinHostPort("localhost", metricsPort)
	log.Printf(" > metrics listening on: [%s]", metricsAddr)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println(http.ListenAndServe(metricsAddr, nil))
	}()

	// netlog backup unix socket
	ctx, cancel := context.WithCancel(context.Background())
	if err := os.MkdirAll(netlog.NetlogUnixSocketAddrDir, os.ModePerm); err != nil {
		log.Errorf("failed to create netlog backup unix socket dir: %s", err)
	} else {
		if addr, err := s.netlogBackupSocketSetup(ctx, netlog.NetlogUnixSocketAddrDir, netlog.NetlogUnixSocketFileName); err != nil {
			log.Errorf("failed to create netlog backup unix socket: %s", err)
		} else {
			log.Debugf("netlog backup unix socket: %s", addr)
		}
	}

	s.instr.GaugeLifeSignal.Set(1)
	defer s.instr.GaugeLifeSignal.Set(0)

	<-chOsInterrupt
	log.Warn("os interrupt received ...")
	// go to sleep 🥱
	s.gracefulShutdown(httpServer, cancel)
}

func (s *Server) gracefulShutdown(httpServer *http.Server, cancel context.CancelFunc) {
	log.Debug("graceful shutdown initiated ...")

	cancel()

	s.board.Close()

	if s.blogApi != nil {
		s.blogApi.CloseDB()
	}

	log.Debugln("removing netlog backup unix socket ...")
	if err := os.RemoveAll(netlog.NetlogUnixSocketAddrDir); err != nil {
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

func (s *Server) corsMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO: fix to allow only "www.serj-tubin.com"

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

// drainAndCloseMiddleware - avoid potential overhead and memory leaks by draining the request body and closing it
func (s *Server) drainAndCloseMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
		})
	}
}

func (s *Server) panicRecoveryMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("http: panic serving %s: %v\n%s", req.URL.Path, r, debug.Stack())
					s.instr.CounterHandleRequestPanic.Inc()
				}
			}()

			// handler call
			next.ServeHTTP(respWriter, req)
		})
	}
}

func (s *Server) metricsMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
			defer func(begin time.Time) {
				s.instr.HistRequestDuration.Observe(time.Since(begin).Seconds())
			}(time.Now())

			// handler call
			next.ServeHTTP(respWriter, req)
		})
	}
}

func (s *Server) connStateMetrics(_ net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		s.instr.CounterRequests.Add(1)
		s.instr.GaugeRequests.Add(1)
	case http.StateClosed:
		s.instr.GaugeRequests.Add(-1)
	}
}

// this is a deliberately overengineered method of communicating of netlog backup with the main service
// just so I have a piece of code that uses UNIX socket interprocess communication, and also to avoid
// adding the Prometheus push gateway to push metrics to it
func (s *Server) netlogBackupSocketSetup(ctx context.Context, socketAddrDir, socketFileName string) (net.Addr, error) {
	socket := filepath.Join(socketAddrDir, socketFileName)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("binding to unix socket %s: %w", socket, err)
	}

	if err := os.Chmod(socket, os.ModeSocket|0666); err != nil {
		return nil, err
	}

	go func() {
		go func() {
			<-ctx.Done()
			_ = listener.Close()
		}()

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("netlog backup unix socket listener conn accept: %s", err)
				return
			}
			log.Debugf("netlog backup unix socket got new conn: %s", conn.RemoteAddr().String())

			// if it takes over 5 minutes to transfer all netlog data, then something is probably not right
			if err := conn.SetDeadline(time.Now().Add(5 * time.Minute)); err != nil {
				log.Errorf("failed to set conn timeout: %s", err)
				return
			}

			go func() {
				defer func() { _ = conn.Close() }()

				buf := make([]byte, 1024)
				n, err := conn.Read(buf)
				if err != nil {
					return
				}

				messageReceived := string(buf[:n])
				log.Infof("netlog backup unix socket received: %s", messageReceived)
				if !strings.HasPrefix(messageReceived, "visits-count::") {
					log.Errorf("netlog backup conn, invalid message received: %s", messageReceived)
					return
				}

				visitsCountStr := strings.Split(messageReceived, "::")[1]
				visitsCount, err := strconv.Atoi(visitsCountStr)
				if err != nil {
					log.Errorf("netlog backup conn, invalid visits counter: %s", err)
					return
				}

				s.instr.CounterVisitsBackups.Add(float64(visitsCount))

				_, err = conn.Write([]byte("ok"))
				if err != nil {
					log.Errorf("netlog backup conn, send response: %s", err)
				}
			}()
		}
	}()

	return listener.Addr(), nil
}
