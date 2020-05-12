package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	quotesManager     *QuotesManager
	openWeatherApiKey string
}

func NewServer(openWeatherApiKey string) *Server {
	s := &Server{
		openWeatherApiKey: openWeatherApiKey,
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
		w.Write([]byte(":)"))
	})

	r.HandleFunc("/quote/random", func(w http.ResponseWriter, r *http.Request) {
		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")
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

	r.HandleFunc("/weather", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if s.openWeatherApiKey == "" {
			log.Errorf("error getting Weather info info: open weather api key not set")
			http.Error(w, "weather api error", http.StatusInternalServerError)
			return
		}

		geoIpInfo, err := getRequestGeoInfo(r)
		if err != nil {
			log.Errorf("error getting geo ip info: %s", err)
			http.Error(w, "geo ip info error", http.StatusInternalServerError)
			return
		}

		weatherInfo, err := getWeatherInfo(geoIpInfo, s.openWeatherApiKey)
		if err != nil {
			log.Errorf("error getting weather info: %s", err)
			http.Error(w, "weather api error", http.StatusInternalServerError)
			return
		}

		var weatherMain []string
		for _, w := range weatherInfo.Weather {
			weatherMain = append(weatherMain, w.Main)
		}

		testResponse := fmt.Sprintf(`{"weather": "%s"}`, strings.Join(weatherMain, ", "))
		_, err = w.Write([]byte(testResponse))
		if err != nil {
			log.Errorf("failed to write response for weather: %s", err)
		}
	})

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
