package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"

	"github.com/coocood/freecache"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
)

// example API call
// http://api.openweathermap.org/data/2.5/weather?q=London,uk&APPID=TODO

const (
	oneHour            = 60 * 60
	weatherCacheExpire = oneHour * 1 // default expire in hours
)

type Api struct {
	cache             *freecache.Cache
	openWeatherApiUrl string // http://api.openweathermap.org/data/2.5/weather
	openWeatherApiKey string
	citiesData        map[string]*[]City
	httpClient        *http.Client
}

func NewApi(openWeatherApiUrl, openWeatherApiKey string, citiesData []City, httpClient *http.Client) *Api {
	megabyte := 1024 * 1024
	cacheSize := 50 * megabyte

	weatherApi := &Api{
		openWeatherApiUrl: openWeatherApiUrl,
		openWeatherApiKey: openWeatherApiKey,
		cache:             freecache.NewCache(cacheSize),
		httpClient:        httpClient,
	}

	loadedCities := 0
	weatherApi.citiesData = make(map[string]*[]City)
	for i := range citiesData {
		loadedCities++
		c := citiesData[i]
		cityName := strings.ToLower(c.Name)
		if cList, ok := weatherApi.citiesData[cityName]; ok {
			*cList = append(*cList, c)
		} else {
			weatherApi.citiesData[cityName] = &[]City{c}
		}
	}

	log.Debugf("loaded %d city names", len(weatherApi.citiesData))
	log.Debugf("total loaded cities: %d", loadedCities)

	return weatherApi
}

func (w *Api) GetWeatherCurrent(ctx context.Context, cityID int, cityName string) (weatherApiResponse *ApiResponse, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "weatherApi.getWeatherCurrent")
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, fmt.Sprintf("found current weeather info for: %s", cityName))
		}
	}()

	// must initialize it, otherwise json.Unmarshal(...) below fails
	// https://stackoverflow.com/questions/20478577/why-does-json-unmarshal-work-with-reference-but-not-pointer
	weatherApiResponse = &ApiResponse{}

	cacheKey := fmt.Sprintf("current::%d", cityID)
	if currentCityWeatherBytes, err := w.cache.Get([]byte(cacheKey)); err == nil {
		log.Tracef("found current weather info for %s in cache", cityName)
		if err = json.Unmarshal(currentCityWeatherBytes, weatherApiResponse); err == nil {
			return weatherApiResponse, nil
		} else {
			log.Errorf("failed to unmarshal current weather from cache for city %s: %s", cityName, err)
		}
	} else {
		log.Debugf("get current weather for city %s from cache: %s; will get the data from open weather api", cityName, err)
	}

	weatherApiUrl := fmt.Sprintf("%s/weather?id=%d&appid=%s", w.openWeatherApiUrl, cityID, w.openWeatherApiKey)
	log.Debugf("calling weather api info: %s", weatherApiUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", weatherApiUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client do: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather api response bytes: %w", err)
	}

	if err := json.Unmarshal(respBytes, weatherApiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal weather api response bytes: %w", err)
	}

	// set cache
	if err = w.cache.Set([]byte(cacheKey), respBytes, weatherCacheExpire); err != nil {
		log.Errorf("failed to write current weather cache for %s %d: %s", cityName, cityID, err)
	} else {
		log.Debugf("current weather cache set for city: %s", cityName)
	}

	return weatherApiResponse, nil
}

// Get5DaysWeatherForecast returns something like sunny, cloudy, etc
func (w *Api) Get5DaysWeatherForecast(ctx context.Context, cityID int, cityName, cityCountry string) (forecast []Info, err error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "weatherApi.get5DaysWeatherForecast")
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, fmt.Sprintf("found 5 days weather info for: %s", cityName))
		}
	}()

	weatherApiResponse := &Api5DaysResponse{}
	cacheKey := fmt.Sprintf("5days::%d", cityID)
	if weatherBytes, err := w.cache.Get([]byte(cacheKey)); err == nil {
		log.Tracef("found 5 days weather info for %s in cache", cityName)
		if err = json.Unmarshal(weatherBytes, weatherApiResponse); err == nil {
			return weatherApiResponse.List, nil
		} else {
			log.Errorf("failed to unmarshal 5 days weather from cache for city %s: %s", cityName, err)
		}
	} else {
		log.Debugf("cached 5 days weather for city %s not found: %s", cityName, err)
	}

	log.Tracef("getting 5 days weather forecast for: %d %s, %s", cityID, cityName, cityCountry)

	// info https://openweathermap.org/forecast5
	weatherApiUrl := fmt.Sprintf("%s/forecast?id=%d&appid=%s&units=metric", w.openWeatherApiUrl, cityID, w.openWeatherApiKey)
	log.Debugf("calling weather api city info: %s", weatherApiUrl)

	req, err := http.NewRequestWithContext(ctx, "GET", weatherApiUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting weather api response: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather api response bytes: %w", err)
	}

	err = json.Unmarshal(respBytes, weatherApiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal weather api 5 days response bytes: %w", err)
	}

	// set cache
	if err = w.cache.Set([]byte(cacheKey), respBytes, weatherCacheExpire); err != nil {
		log.Errorf("failed to write 5 days weather for %s %d: %s", cityName, cityID, err)
	} else {
		log.Debugf("5 days weather cache set for city: %s", cityName)
	}

	return weatherApiResponse.List, nil
}

func (w *Api) GetWeatherCity(city, countryCode string) (*City, error) {
	cityName := strings.ToLower(city)
	log.Debugf("weather-api: get weather city for: %s", cityName)

	citiesList, found := w.citiesData[cityName]
	if !found {
		return nil, ErrNotFound
	}

	log.Debugf("weather-api: get weather city for: %s, found %d cities", cityName, len(*citiesList))

	if len(*citiesList) == 1 {
		return &(*citiesList)[0], nil
	}

	country := strings.ToLower(countryCode)
	for i := range *citiesList {
		c := (*citiesList)[i]
		if strings.ToLower(c.Country) == country {
			log.Debugf("weather-api: found weather city: %s / %s", c.Name, c.Country)
			return &c, nil
		}
	}

	log.Debugf("weather-api: get weather city for: %s, nothing found in the end ...", cityName)

	return nil, ErrNotFound
}
