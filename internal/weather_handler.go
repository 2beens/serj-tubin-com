package internal

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type WeatherHandler struct {
	geoIp             *GeoIp
	weatherApi        *WeatherApi
	openWeatherApiKey string
}

var (
	ErrNotFound = errors.New("not found")
)

func NewWeatherHandler(weatherRouter *mux.Router, geoIp *GeoIp, weatherApi *WeatherApi, openWeatherApiKey string) *WeatherHandler {
	handler := &WeatherHandler{
		openWeatherApiKey: openWeatherApiKey,
		geoIp:             geoIp,
		weatherApi:        weatherApi,
	}

	weatherRouter.HandleFunc("/current", handler.handleCurrent).Methods("GET")
	weatherRouter.HandleFunc("/tomorrow", handler.handleTomorrow).Methods("GET")
	weatherRouter.HandleFunc("/5days", handler.handle5Days).Methods("GET")

	return handler
}

func (handler *WeatherHandler) handleCurrent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if handler.openWeatherApiKey == "" {
		log.Errorf("error getting Weather info: open weather api key not set")
		http.Error(w, "weather api error", http.StatusInternalServerError)
		return
	}

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	city, err := handler.weatherApi.GetWeatherCity(geoIpInfo)
	if err != nil {
		log.Errorf("error getting current weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.GetWeatherCurrent(city, handler.openWeatherApiKey)
	if err != nil {
		log.Errorf("error getting weather info: %s", err)
		http.Error(w, "weather api error", http.StatusInternalServerError)
		return
	}

	weatherDescriptionsBytes, err := json.Marshal(weatherInfo.WeatherDescriptions)
	if err != nil {
		log.Errorf("error marshaling weather descriptions for %s: %s", city.Name, err)
		http.Error(w, "weather api marshal error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(weatherDescriptionsBytes)
	if err != nil {
		log.Errorf("failed to write response for weather: %s", err)
	}
}

func (handler *WeatherHandler) handleTomorrow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if handler.openWeatherApiKey == "" {
		log.Errorf("error getting Weather info: open weather api key not set")
		http.Error(w, "weather api error", http.StatusInternalServerError)
		return
	}

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	city, err := handler.weatherApi.GetWeatherCity(geoIpInfo)
	if err != nil {
		log.Errorf("error getting weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.Get5DaysWeatherForecast(city, handler.openWeatherApiKey)
	if err != nil {
		log.Errorf("error getting weather tomorrow info: %s", err)
		http.Error(w, "weather tomorrow error", http.StatusInternalServerError)
		return
	}

	tomorrow := time.Now().Add(24 * time.Hour)
	var weatherForecast []WeatherInfoShort
	for _, w := range weatherInfo {
		wt := w.Timestamp()
		if wt.Day() == tomorrow.Day() && wt.Month() == tomorrow.Month() && wt.Year() == tomorrow.Year() {
			weatherForecast = append(weatherForecast, WeatherInfoShort{
				Timestamp:           w.Dt,
				WeatherDescriptions: w.WeatherDescriptions,
			})
		}
	}

	weatherForecastBytes, err := json.Marshal(weatherForecast)
	if err != nil {
		log.Errorf("failed to unmarshal weather forecast for tomorrow for %s: %s", city.Name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(weatherForecastBytes)
	if err != nil {
		log.Errorf("failed to write response for weather tomorrow: %s", err)
	}
}

func (handler *WeatherHandler) handle5Days(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if handler.openWeatherApiKey == "" {
		log.Errorf("error getting Weather info: open weather api key not set")
		http.Error(w, "weather api error", http.StatusInternalServerError)
		return
	}

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	city, err := handler.weatherApi.GetWeatherCity(geoIpInfo)
	if err != nil {
		log.Errorf("error getting weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.Get5DaysWeatherForecast(city, handler.openWeatherApiKey)
	if err != nil {
		log.Errorf("error getting weather tomorrow info: %s", err)
		http.Error(w, "weather tomorrow error", http.StatusInternalServerError)
		return
	}

	var weatherForecast []WeatherInfoShort
	for _, w := range weatherInfo {
		weatherForecast = append(weatherForecast, WeatherInfoShort{
			Timestamp:           w.Dt,
			WeatherDescriptions: w.WeatherDescriptions,
		})
	}

	weatherForecastBytes, err := json.Marshal(weatherForecast)
	if err != nil {
		log.Errorf("failed to unmarshal weather 5 days forecast for %s: %s", city.Name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(weatherForecastBytes)
	if err != nil {
		log.Errorf("failed to write response for weather tomorrow: %s", err)
	}
}
