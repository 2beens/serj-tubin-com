package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/coocood/freecache"
	log "github.com/sirupsen/logrus"
)

// example API call
// http://api.openweathermap.org/data/2.5/weather?q=London,uk&APPID=0af09f7bce2fd9cbea44d6740f3c8e27

type WeatherApi struct {
	cache *freecache.Cache
}

func NewWeatherApi(cacheSizeMegabytes int) *WeatherApi {
	megabyte := 1024 * 1024
	cacheSize := cacheSizeMegabytes * megabyte

	return &WeatherApi{
		cache: freecache.NewCache(cacheSize),
	}
}

func (w *WeatherApi) GetWeatherCurrent(city WeatherCity, weatherApiKey string) (WeatherApiResponse, error) {
	weatherApiResponse := &WeatherApiResponse{}

	cacheKey := fmt.Sprintf("current::%s", city.ID)
	if currentCityWeatherBytes, err := w.cache.Get([]byte(cacheKey)); err == nil {
		if err = json.Unmarshal(currentCityWeatherBytes, weatherApiResponse); err == nil {
			return *weatherApiResponse, nil
		} else {
			log.Errorf("failed to unmarshal current weather from cache for city %s: %s", city.Name, err)
		}
	} else {
		log.Debugf("cached current weather for city %s not found: %s", city.Name, err)
	}

	weatherApiUrl := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?id=%d&appid=%s", city.ID, weatherApiKey)
	log.Debugf("calling weather api info: %s", weatherApiUrl)

	resp, err := http.Get(weatherApiUrl)
	if err != nil {
		return WeatherApiResponse{}, fmt.Errorf("error getting weather api response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return WeatherApiResponse{}, fmt.Errorf("failed to read weather api response bytes: %s", err)
	}

	err = json.Unmarshal(respBytes, weatherApiResponse)
	if err != nil {
		return WeatherApiResponse{}, fmt.Errorf("failed to unmarshal weather api response bytes: %s", err)
	}

	// set cache
	if err = w.cache.Set([]byte(cacheKey), respBytes, WeatherCacheExpire); err != nil {
		log.Errorf("failed to write geo ip cache for %s %d: %s", city.Name, city.ID, err)
	} else {
		log.Debugf("geo ip cache set for city: %s", city.Name)
	}

	return *weatherApiResponse, nil
}

// returns something like sunny, cloudy, etc
func (w *WeatherApi) GetWeatherTomorrow(city WeatherCity, weatherApiKey string) ([]string, error) {
	log.Tracef("getting weather tomorrow for: %d %s / $s", city.ID, city.Name, city.Country)

	//  get city ID and make a open weather API call to get weather for tomorrow

	weatherApiUrl := fmt.Sprintf("http://api.openweathermap.org/data/2.5/forecast?id=%d&appid=%s", city.ID, weatherApiKey)
	log.Debugf("calling weather api city info: %s", weatherApiUrl)

	resp, err := http.Get(weatherApiUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting weather api response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read weather api response bytes: %s", err)
	}

	weatherApiResponse := &WeatherApi5DaysResponse{}
	err = json.Unmarshal(respBytes, weatherApiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal weather api 5 days response bytes: %s", err)
	}

	var weatherTomorrow []string
	for _, wi := range weatherApiResponse.List {
		for _, w := range wi.Weather {
			weatherTomorrow = append(weatherTomorrow, w.Main)
		}
	}

	return weatherTomorrow, nil
}

func (w *WeatherApi) LoadCitiesData(cityListDataPath string) ([]WeatherCity, error) {
	citiesJsonFile, err := os.Open(cityListDataPath)
	if err != nil {
		return []WeatherCity{}, err
	}

	citiesJsonFileData, err := ioutil.ReadAll(citiesJsonFile)
	if err != nil {
		return []WeatherCity{}, err
	}

	var cities []WeatherCity
	err = json.Unmarshal(citiesJsonFileData, &cities)
	if err != nil {
		return []WeatherCity{}, err
	}

	return cities, nil
}
