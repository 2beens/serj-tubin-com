package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	quotesManager *QuotesManager
}

func NewServer() *Server {
	s := &Server{}

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

	r.HandleFunc("/weather/tomorrow", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// allowed up to 15,000 queries per hour
		// https://freegeoip.app/
		userIp, err := ReadUserIP(r)
		if err != nil {
			log.Errorf("error getting user ip: %s", err.Error())
			http.Error(w, "geoip error", http.StatusInternalServerError)
			return
		}

		geoIpUrl := fmt.Sprintf("https://freegeoip.app/json/%s", userIp)
		log.Debugf("calling geo ip info: %s", geoIpUrl)

		resp, err := http.Get(geoIpUrl)
		if err != nil {
			log.Errorf("error getting freegeoip response: %s", err.Error())
			http.Error(w, "geoip error", http.StatusInternalServerError)
			return
		}

		respBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to read geo ip response bytes: %s", err)
			http.Error(w, "geoip error", http.StatusInternalServerError)
			return
		}

		geoIpResponse := &GeoIpResponse{}
		err = json.Unmarshal(respBytes, geoIpResponse)
		if err != nil {
			log.Errorf("failed to unmarshal geo ip response bytes: %s", err)
			http.Error(w, "geoip error", http.StatusInternalServerError)
			return
		}

		testResponse := fmt.Sprintf(`{"city": "%s", "country":"%s", "country_code": "%s"}`, geoIpResponse.City, geoIpResponse.CountryName, geoIpResponse.CountryCode)
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
