package weather

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/2beens/serjtubincom/internal/geoip"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"

	log "github.com/sirupsen/logrus"
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

func NewHandler(
	geoIp *geoip.Api,
	weatherApi *Api,
) *Handler {
	return &Handler{
		geoIp:      geoIp,
		weatherApi: weatherApi,
	}
}

func (handler *Handler) HandleCurrent(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "weather.handleCurrent")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	userIp, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	locationInfo, err := handler.geoIp.GetIPGeoInfo(ctx, userIp)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("error getting geo ip info: %s", err))
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	log.Debugf("weather-handler: handle current for city [%s] and country code [%s]", locationInfo.City, locationInfo.Country)
	span.SetAttributes(attribute.String("city", locationInfo.City))
	span.SetAttributes(attribute.String("country", locationInfo.Country))

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City, locationInfo.Country)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf(
			"get current weather city from geo ip info for city [%s] and country code [%s]: %s",
			locationInfo.City, locationInfo.Country, err,
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

func (handler *Handler) HandleTomorrow(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "weather.handleTomorrow")
	defer span.End()

	// TODO: this is not respected
	w.Header().Set("Content-Type", "application/json")

	userIp, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	locationInfo, err := handler.geoIp.GetIPGeoInfo(ctx, userIp)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	log.Debugf("weather-handler: handle tomorrow weather for city [%s] and country code [%s]", locationInfo.City, locationInfo.Country)
	span.SetAttributes(attribute.String("city", locationInfo.City))
	span.SetAttributes(attribute.String("country", locationInfo.Country))

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City, locationInfo.Country)
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

func (handler *Handler) Handle5Days(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.GlobalTracer.Start(r.Context(), "weather.handle5Days")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")

	userIp, err := pkg.ReadUserIP(r)
	if err != nil {
		span.SetStatus(codes.Error, fmt.Sprintf("get user ip: %s", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	locationInfo, err := handler.geoIp.GetIPGeoInfo(ctx, userIp)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		log.Errorf("error getting geo ip info: %s", err)
		http.Error(w, "geo ip info error", http.StatusInternalServerError)
		return
	}

	log.Debugf("weather-handler: handle 5 days weather for city [%s] and country code [%s]", locationInfo.City, locationInfo.Country)
	span.SetAttributes(attribute.String("city", locationInfo.City))
	span.SetAttributes(attribute.String("country", locationInfo.Country))

	city, err := handler.weatherApi.GetWeatherCity(locationInfo.City, locationInfo.Country)
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
