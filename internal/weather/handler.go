package weather

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/2beens/serjtubincom/internal/geoip"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Handler struct {
	geoIp      *geoip.Api
	weatherApi *Api
}

var (
	ErrNotFound = errors.New("not found")
)

// TODO: refactor this old handler;
//	- loading cities data from a file should not be done here
//  - weather api should be injected (and also unit tests added)
//	- add related changes to the previous one

func NewHandler(weatherRouter *mux.Router, geoIp *geoip.Api, openWeatherAPIUrl, openWeatherApiKey string) (*Handler, error) {
	citiesData, err := LoadCitiesData("./assets/city.list.json")
	if err != nil {
		log.Errorf("failed to load weather cities data: %s", err)
		return nil, fmt.Errorf("failed to load weather cities data: %s", err)
	}

	// TODO: again - refactor this, like described above
	tracedHttpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	handler := &Handler{
		geoIp: geoIp,
		weatherApi: NewApi(
			openWeatherAPIUrl,
			openWeatherApiKey,
			citiesData,
			tracedHttpClient,
		),
	}

	weatherRouter.HandleFunc("/current", handler.handleCurrent).Methods("GET")
	weatherRouter.HandleFunc("/tomorrow", handler.handleTomorrow).Methods("GET")
	weatherRouter.HandleFunc("/5days", handler.handle5Days).Methods("GET")

	return handler, nil
}

func LoadCitiesData(cityListDataPath string) ([]City, error) {
	citiesJsonFile, err := os.Open(cityListDataPath)
	if err != nil {
		return []City{}, err
	}

	citiesJsonFileData, err := io.ReadAll(citiesJsonFile)
	if err != nil {
		return []City{}, err
	}

	var cities []City
	err = json.Unmarshal(citiesJsonFileData, &cities)
	if err != nil {
		return []City{}, err
	}

	return cities, nil
}

func (handler *Handler) handleCurrent(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "weather.handleCurrent")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	userIp, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(ctx, userIp)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("error getting geo ip info: %s", err))
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	locationInfo := geoIpInfo.Data.Location
	log.Debugf("weather-handler: handle current for city [%s] and country code [%s]", locationInfo.City.Name, locationInfo.Country.Name)
	span.SetAttributes(attribute.String("city", locationInfo.City.Name))
	span.SetAttributes(attribute.String("country", locationInfo.Country.Name))

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City.Name, locationInfo.Country.Alpha2)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf(
			"get current weather city from geo ip info for city [%s] and country code [%s]: %s",
			locationInfo.City.Name, locationInfo.Country.Alpha2, err,
		)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.GetWeatherCurrent(ctx, city.ID, city.Name)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error getting weather info: %s", err)
		http.Error(w, "weather api error", http.StatusInternalServerError)
		return
	}

	weatherDescriptionsBytes, err := json.Marshal(weatherInfo.WeatherDescriptions)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error marshaling weather descriptions for %s: %s", city.Name, err)
		http.Error(w, "weather api marshal error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(weatherDescriptionsBytes)
	if err != nil {
		log.Errorf("failed to write response for weather: %s", err)
	}
}

func (handler *Handler) handleTomorrow(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "weather.handleTomorrow")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	userIp, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(ctx, userIp)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	locationInfo := geoIpInfo.Data.Location
	log.Debugf("weather-handler: handle tomorrow weather for city [%s] and country code [%s]", locationInfo.City.Name, locationInfo.Country.Name)
	span.SetAttributes(attribute.String("city", locationInfo.City.Name))
	span.SetAttributes(attribute.String("country", locationInfo.Country.Name))

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City.Name, locationInfo.Country.Alpha2)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("handle weather tomorrow: error getting weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.Get5DaysWeatherForecast(ctx, city.ID, city.Name, city.Country)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error getting weather tomorrow info: %s", err)
		http.Error(w, "weather tomorrow error", http.StatusInternalServerError)
		return
	}

	tomorrow := time.Now().Add(24 * time.Hour)
	var weatherForecast []InfoShort
	for _, w := range weatherInfo {
		wt := w.Timestamp()
		if wt.Day() == tomorrow.Day() && wt.Month() == tomorrow.Month() && wt.Year() == tomorrow.Year() {
			weatherForecast = append(weatherForecast, InfoShort{
				Timestamp:           w.Dt,
				WeatherDescriptions: w.WeatherDescriptions,
			})
		}
	}

	weatherForecastBytes, err := json.Marshal(weatherForecast)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("failed to unmarshal weather forecast for tomorrow for %s: %s", city.Name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(weatherForecastBytes)
	if err != nil {
		log.Errorf("failed to write response for weather tomorrow: %s", err)
	}
}

func (handler *Handler) handle5Days(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "weather.handle5Days")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	userIp, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	geoIpInfo, err := handler.geoIp.GetRequestGeoInfo(ctx, userIp)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	locationInfo := geoIpInfo.Data.Location
	log.Debugf("weather-handler: handle 5 days weather for city [%s] and country code [%s]", locationInfo.City.Name, locationInfo.Country.Name)
	span.SetAttributes(attribute.String("city", locationInfo.City.Name))
	span.SetAttributes(attribute.String("country", locationInfo.Country.Name))

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City.Name, locationInfo.Country.Alpha2)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("handle weather 5 days: error getting weather city from geo ip info: %s", err)
		http.Error(w, "weather city info error", http.StatusInternalServerError)
		return
	}

	weatherInfo, err := handler.weatherApi.Get5DaysWeatherForecast(ctx, city.ID, city.Name, city.Country)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error getting weather tomorrow info: %s", err)
		http.Error(w, "weather tomorrow error", http.StatusInternalServerError)
		return
	}

	var weatherForecast []InfoShort
	for _, w := range weatherInfo {
		weatherForecast = append(weatherForecast, InfoShort{
			Timestamp:           w.Dt,
			WeatherDescriptions: w.WeatherDescriptions,
		})
	}

	weatherForecastBytes, err := json.Marshal(weatherForecast)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("failed to unmarshal weather 5 days forecast for %s: %s", city.Name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(weatherForecastBytes)
	if err != nil {
		log.Errorf("failed to write response for weather tomorrow: %s", err)
	}
}
