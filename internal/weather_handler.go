package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type WeatherHandler struct {
	geoIp      *GeoIp
	weatherApi *WeatherApi
}

var (
	ErrNotFound = errors.New("not found")
)

func NewWeatherHandler(weatherRouter *mux.Router, geoIp *GeoIp, openWeatherAPIUrl, openWeatherApiKey string) (*WeatherHandler, error) {
	citiesData, err := LoadCitiesData("./assets/city.list.json")
	if err != nil {
		log.Errorf("failed to load weather cities data: %s", err)
		return nil, fmt.Errorf("failed to load weather cities data: %s", err)
	}

	handler := &WeatherHandler{
		geoIp: geoIp,
		weatherApi: NewWeatherApi(
			openWeatherAPIUrl,
			openWeatherApiKey,
			citiesData,
			http.DefaultClient,
		),
	}

	weatherRouter.HandleFunc("/current", handler.handleCurrent).Methods("GET")
	weatherRouter.HandleFunc("/tomorrow", handler.handleTomorrow).Methods("GET")
	weatherRouter.HandleFunc("/5days", handler.handle5Days).Methods("GET")

	return handler, nil
}

func (handler *WeatherHandler) handleCurrent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	locationInfo := geoIpInfo.Data.Location
	log.Debugf("weather-handler: handle current for city [%s] and country code [%s]", locationInfo.City.Name, locationInfo.Country.Name)

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City.Name, locationInfo.Country.Alpha2)
	if err != nil {
		log.Errorf("error getting current weather city from geo ip info for city [%s] and country code [%s]: %s", err, locationInfo.City.Name, locationInfo.Country.Alpha2)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.GetWeatherCurrent(city.ID, city.Name)
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

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	locationInfo := geoIpInfo.Data.Location
	log.Debugf("weather-handler: handle tomorrow weather for city [%s] and country code [%s]", locationInfo.City.Name, locationInfo.Country.Name)

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City.Name, locationInfo.Country.Alpha2)
	if err != nil {
		log.Errorf("handle weather tomorrow: error getting weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.Get5DaysWeatherForecast(city.ID, city.Name, city.Country)
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

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	locationInfo := geoIpInfo.Data.Location
	log.Debugf("weather-handler: handle 5 days weather for city [%s] and country code [%s]", locationInfo.City.Name, locationInfo.Country.Name)

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City.Name, locationInfo.Country.Alpha2)
	if err != nil {
		log.Errorf("handle weather 5 days: error getting weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.Get5DaysWeatherForecast(city.ID, city.Name, city.Country)
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
