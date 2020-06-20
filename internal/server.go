package internal

import (
	"context"
	"encoding/json"
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
	weatherApi    *WeatherApi
	quotesManager *QuotesManager
	board         *Board

	openWeatherApiKey string
	muteRequestLogs   bool
}

func NewServer(aerospikeHost string, aerospikePort int, aeroBoardNamespace, openWeatherApiKey string) *Server {
	board, err := NewBoard(aerospikeHost, aerospikePort, aeroBoardNamespace)
	if err != nil {
		log.Errorf("failed to create visitor board: %s", err)
	}

	s := &Server{
		openWeatherApiKey: openWeatherApiKey,
		muteRequestLogs:   false,
		geoIp:             NewGeoIp(50),
		weatherApi:        NewWeatherApi(50, "./assets/city.list.json"),
		board:             board,
	}

	qm, err := NewQuoteManager("./assets/quotes.csv")
	if err != nil {
		panic(err)
	}

	s.quotesManager = qm

	return s
}

func (s *Server) routerSetup() (r *mux.Router) {
	r = mux.NewRouter()

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

	weatherRouter := r.PathPrefix("/weather").Subrouter()
	boardRouter := r.PathPrefix("/board").Subrouter()
	NewWeatherHandler(weatherRouter, s.geoIp, s.weatherApi, s.openWeatherApiKey)
	NewBoardHandler(boardRouter, s.board)

	r.Use(s.corsMiddleware())
	r.Use(s.loggingMiddleware())

	return r
}

func (s *Server) Serve(port int) {
	router := s.routerSetup()

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
