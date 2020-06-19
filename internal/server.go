package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	OneHour            = 60 * 60
	GeoIpCacheExpire   = OneHour * 5 // default expire in hours
	WeatherCacheExpire = OneHour * 1 // default expire in hours

	// TODO: put in config
	AerospikeBoardNamespace = "board"
)

type Server struct {
	geoIp         *GeoIp
	weatherApi    *WeatherApi
	quotesManager *QuotesManager
	board         *Board

	openWeatherApiKey string
	muteRequestLogs   bool
}

func NewServer(aerospikeHost string, aerospikePort int, openWeatherApiKey string) *Server {
	board, err := NewBoard(aerospikeHost, aerospikePort, AerospikeBoardNamespace)
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

	// TODO: add board router instead
	r.HandleFunc("/board/messages/new", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		}

		err := r.ParseForm()
		if err != nil {
			log.Errorf("add new message failed, parse form error: %s", err)
			w.Write([]byte("error 500: parse form error"))
			return
		}

		message := r.Form.Get("message")
		author := r.Form.Get("author")
		timestamp := time.Now().Unix()

		err = s.board.StoreMessage(BoardMessage{
			Author:    author,
			Timestamp: timestamp,
			Message:   message,
		})

		if err != nil {
			log.Errorf("store new message error: %s", err)
			w.Write([]byte("error 500: get messages error"))
			return
		}

		w.Write([]byte("added <3"))
	})

	r.HandleFunc("/board/messages/all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "only GET allowed", http.StatusMethodNotAllowed)
		}

		boardMessages, err := s.board.AllMessages()
		if err != nil {
			log.Errorf("get all messages error: %s", err)
			w.Write([]byte("error 500: get messages error"))
			return
		}

		messagesJson, err := json.Marshal(boardMessages)
		if err != nil {
			log.Errorf("marshal all messages error: %s", err)
			w.Write([]byte("error 500: get messages error"))
			return
		}

		w.Write(messagesJson)
	})

	weatherRouter := r.PathPrefix("/weather").Subrouter()
	NewWeatherHandler(weatherRouter, s.geoIp, s.weatherApi, s.openWeatherApiKey)

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

	log.Infof(" > server listening on: [%s]", ipAndPort)
	log.Fatal(httpServer.ListenAndServe())
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
