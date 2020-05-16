package internal

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type WeatherHandler struct {
	geoIp             *GeoIp
	weatherApi        *WeatherApi
	openWeatherApiKey string
	citiesData        map[string]*[]WeatherCity
}

var (
	ErrNotFound = errors.New("not found")
)

func NewWeatherHandler(weatherRouter *mux.Router, geoIp *GeoIp, weatherApi *WeatherApi, citiesDataPath, openWeatherApiKey string) *WeatherHandler {
	handler := &WeatherHandler{
		openWeatherApiKey: openWeatherApiKey,
		geoIp:             geoIp,
		weatherApi:        weatherApi,
	}

	loadedCities := 0
	citiesData, err := weatherApi.LoadCitiesData(citiesDataPath)
	if err != nil {
		log.Errorf("failed to load weather cities data: %s", err)
	} else {
		handler.citiesData = make(map[string]*[]WeatherCity)
		for i, _ := range citiesData {
			loadedCities++
			c := citiesData[i]
			cityName := strings.ToLower(c.Name)
			if cList, ok := handler.citiesData[cityName]; ok {
				*cList = append(*cList, c)
			} else {
				handler.citiesData[cityName] = &[]WeatherCity{c}
			}
		}
	}

	log.Debugf("loaded %d city names", len(handler.citiesData))
	log.Debugf("total loaded cities: %d", loadedCities)

	weatherRouter.HandleFunc("/tomorrow", handler.handleTomorrow).Methods("GET")
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

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	city, err := handler.getWeatherCity(geoIpInfo)
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

func (handler *WeatherHandler) handleTomorrow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if handler.openWeatherApiKey == "" {
		log.Errorf("error getting Weather info info: open weather api key not set")
		http.Error(w, "weather api error", http.StatusInternalServerError)
		return
	}

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(r)
	if err != nil {
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	city, err := handler.getWeatherCity(geoIpInfo)
	if err != nil {
		log.Errorf("error getting weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.GetWeatherTomorrow(city, handler.openWeatherApiKey)
	if err != nil {
		log.Errorf("error getting weather tomorrow info: %s", err)
		http.Error(w, "weather tomorrow error", http.StatusInternalServerError)
		return
	}

	weatherInfoResp := fmt.Sprintf(`{"weather": "%s"}`, strings.Join(weatherInfo, ", "))
	_, err = w.Write([]byte(weatherInfoResp))
	if err != nil {
		log.Errorf("failed to write response for weather tomorrow: %s", err)
	}
}

// TODO: move this to weather api struct
func (handler *WeatherHandler) getWeatherCity(geoInfo *GeoIpInfo) (WeatherCity, error) {
	cityName := strings.ToLower(geoInfo.City)
	citiesList, found := handler.citiesData[cityName]
	if !found {
		return WeatherCity{}, ErrNotFound
	}

	if len(*citiesList) == 1 {
		return (*citiesList)[0], nil
	}

	country := strings.ToLower(geoInfo.CountryCode)
	for i, _ := range *citiesList {
		c := (*citiesList)[i]
		if strings.ToLower(c.Country) == country {
			return c, nil
		}
	}

	return WeatherCity{}, ErrNotFound
}
