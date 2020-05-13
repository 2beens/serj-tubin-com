package internal

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type WeatherHandler struct {
	openWeatherApiKey string
	citiesData        []WeatherCity
}

func NewWeatherHandler(weatherRouter *mux.Router, citiesDataPath, openWeatherApiKey string) *WeatherHandler {
	handler := &WeatherHandler{
		openWeatherApiKey: openWeatherApiKey,
	}

	citiesData, err := loadCitiesData(citiesDataPath)
	if err != nil {
		log.Errorf("failed to load weather cities data: %s", err)
	}

	handler.citiesData = citiesData
	log.Debugf("loaded %d cities", len(handler.citiesData))

	weatherRouter.HandleFunc("/current", handler.handleCurrent).Methods("GET")

	return handler
}

func (handler *WeatherHandler) handleCurrent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if handler.openWeatherApiKey == "" {
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

	weatherInfo, err := getWeatherInfo(geoIpInfo, handler.openWeatherApiKey)
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
}
