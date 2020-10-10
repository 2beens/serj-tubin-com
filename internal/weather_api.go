package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/coocood/freecache"
	log "github.com/sirupsen/logrus"
)

// example API call
// http://api.openweathermap.org/data/2.5/weather?q=London,uk&APPID=0af09f7bce2fd9cbea44d6740f3c8e27

type WeatherApi struct {
	cache             *freecache.Cache
	openWeatherApiUrl string // http://api.openweathermap.org/data/2.5/weather
	openWeatherApiKey string
	citiesData        map[string]*[]WeatherCity
}

func NewWeatherApi(openWeatherApiUrl, openWeatherApiKey string, citiesData []WeatherCity) *WeatherApi {
	megabyte := 1024 * 1024
	cacheSize := 50 * megabyte

	weatherApi := &WeatherApi{
		openWeatherApiUrl: openWeatherApiUrl,
		openWeatherApiKey: openWeatherApiKey,
		cache:             freecache.NewCache(cacheSize),
	}

	loadedCities := 0
	weatherApi.citiesData = make(map[string]*[]WeatherCity)
	for i := range citiesData {
		loadedCities++
		c := citiesData[i]
		cityName := strings.ToLower(c.Name)
		if cList, ok := weatherApi.citiesData[cityName]; ok {
			*cList = append(*cList, c)
		} else {
			weatherApi.citiesData[cityName] = &[]WeatherCity{c}
		}
	}

	log.Debugf("loaded %d city names", len(weatherApi.citiesData))
	log.Debugf("total loaded cities: %d", loadedCities)

	return weatherApi
}

func (w *WeatherApi) GetWeatherCurrent(cityID int, cityName string) (*WeatherApiResponse, error) {
	weatherApiResponse := &WeatherApiResponse{}

	cacheKey := fmt.Sprintf("current::%d", cityID)
	if currentCityWeatherBytes, err := w.cache.Get([]byte(cacheKey)); err == nil {
		log.Tracef("found current weather info for %s in cache", cityName)
		if err = json.Unmarshal(currentCityWeatherBytes, weatherApiResponse); err == nil {
			return weatherApiResponse, nil
		} else {
			log.Errorf("failed to unmarshal current weather from cache for city %s: %s", cityName, err)
		}
	} else {
		log.Debugf("cached current weather for city %s not found: %s", cityName, err)
	}

	weatherApiUrl := fmt.Sprintf("%s?id=%d&appid=%s", w.openWeatherApiUrl, cityID, w.openWeatherApiKey)
	log.Debugf("calling weather api info: %s", weatherApiUrl)

	resp, err := http.Get(weatherApiUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting weather api response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather api response bytes: %s", err)
	}

	err = json.Unmarshal(respBytes, weatherApiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal weather api response bytes: %s", err)
	}

	// set cache
	if err = w.cache.Set([]byte(cacheKey), respBytes, WeatherCacheExpire); err != nil {
		log.Errorf("failed to write current weather cache for %s %d: %s", cityName, cityID, err)
	} else {
		log.Debugf("current weather cache set for city: %s", cityName)
	}

	return weatherApiResponse, nil
}

// returns something like sunny, cloudy, etc
func (w *WeatherApi) Get5DaysWeatherForecast(city *WeatherCity, weatherApiKey string) ([]WeatherInfo, error) {
	weatherApiResponse := &WeatherApi5DaysResponse{}

	cacheKey := fmt.Sprintf("5days::%d", city.ID)
	if weatherBytes, err := w.cache.Get([]byte(cacheKey)); err == nil {
		log.Tracef("found 5 days weather info for %s in cache", city.Name)
		if err = json.Unmarshal(weatherBytes, weatherApiResponse); err == nil {
			return weatherApiResponse.List, nil
		} else {
			log.Errorf("failed to unmarshal 5 days weather from cache for city %s: %s", city.Name, err)
		}
	} else {
		log.Debugf("cached 5 days weather for city %s not found: %s", city.Name, err)
	}

	log.Tracef("getting 5 days weather forecast for: %d %s, %s", city.ID, city.Name, city.Country)

	// info https://openweathermap.org/forecast5
	weatherApiUrl := fmt.Sprintf("http://api.openweathermap.org/data/2.5/forecast?id=%d&appid=%s&units=metric", city.ID, weatherApiKey)
	log.Debugf("calling weather api city info: %s", weatherApiUrl)

	resp, err := http.Get(weatherApiUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting weather api response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather api response bytes: %s", err)
	}

	err = json.Unmarshal(respBytes, weatherApiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal weather api 5 days response bytes: %s", err)
	}

	// set cache
	if err = w.cache.Set([]byte(cacheKey), respBytes, WeatherCacheExpire); err != nil {
		log.Errorf("failed to write 5 days weather for %s %d: %s", city.Name, city.ID, err)
	} else {
		log.Debugf("5 days weather cache set for city: %s", city.Name)
	}

	return weatherApiResponse.List, nil
}

func (w *WeatherApi) GetWeatherCity(city, countryCode string) (*WeatherCity, error) {
	cityName := strings.ToLower(city)
	citiesList, found := w.citiesData[cityName]
	if !found {
		return nil, ErrNotFound
	}

	if len(*citiesList) == 1 {
		return &(*citiesList)[0], nil
	}

	country := strings.ToLower(countryCode)
	for i := range *citiesList {
		c := (*citiesList)[i]
		if strings.ToLower(c.Country) == country {
			return &c, nil
		}
	}

	return nil, ErrNotFound
}
