package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	OneHour            = 60 * 60
	GeoIpCacheExpire   = OneHour * 5 // default expire in hours
	WeatherCacheExpire = OneHour * 1 // default expire in hours
)

type Server struct {
	geoIp         *GeoIp
	quotesManager *QuotesManager
	board         *Board

	openWeatherAPIUrl string
	openWeatherApiKey string
	muteRequestLogs   bool
}

func NewServer(aerospikeHost string, aerospikePort int, aeroNamespace, openWeatherApiKey string) (*Server, error) {
	board, err := NewBoard(aerospikeHost, aerospikePort, aeroNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create visitor board: %s", err)
	}

	if openWeatherApiKey == "" {
		log.Errorf("error getting Weather info: open weather api key not set")
		return nil, errors.New("open weather API key not set")
	}

	s := &Server{
		openWeatherAPIUrl: "http://api.openweathermap.org/data/2.5",
		openWeatherApiKey: openWeatherApiKey,
		muteRequestLogs:   false,
		geoIp:             NewGeoIp(),
		board:             board,
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

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("I'm OK, thanks :)"))
	})

	r.HandleFunc("/quote/random", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		q := s.quotesManager.RandomQuote()
		qBytes, err := json.Marshal(q)
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
			log.Errorf("marshal quote error: %s", err)
			return
		}

		w.Write(qBytes)
	})

	r.HandleFunc("/whereami", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		geoIpInfo, err := s.geoIp.GetRequestGeoInfo(r)
		if err != nil {
			log.Errorf("error getting geo ip info: %s", err)
			http.Error(w, "geo ip info error", http.StatusInternalServerError)
			return
		}

		geoResp := fmt.Sprintf(`{"city":"%s", "country":"%s"}`, geoIpInfo.City, geoIpInfo.CountryName)
		w.Write([]byte(geoResp))
	})

	// TODO: move in util handler
	r.HandleFunc("/myip", func(w http.ResponseWriter, r *http.Request) {
		ip, err := s.geoIp.ReadUserIP(r)
		if err != nil {
			log.Errorf("failed to get user IP address: %s", err)
			http.Error(w, "failed to get IP", http.StatusInternalServerError)
		}
		w.Write([]byte(ip))
	})

	// TODO: maybe add version info, to return it to site pages header

	weatherRouter := r.PathPrefix("/weather").Subrouter()
	boardRouter := r.PathPrefix("/board").Subrouter()

	if NewBoardHandler(boardRouter, s.board) == nil {
		return nil, errors.New("board handler is nil")
	}

	if err, weatherHandler := NewWeatherHandler(weatherRouter, s.geoIp, s.openWeatherAPIUrl, s.openWeatherApiKey); err != nil {
		return nil, fmt.Errorf("failed to create weather handler: %w", err)
	} else if weatherHandler == nil {
		return nil, errors.New("weather handler is nil")
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

	select {
	case <-chOsInterrupt:
		log.Warn("os interrupt received ...")
	}
	s.gracefulShutdown(httpServer)
}

func (s *Server) gracefulShutdown(httpServer *http.Server) {
	log.Debug("graceful shutdown initiated ...")

	s.board.Close()

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
