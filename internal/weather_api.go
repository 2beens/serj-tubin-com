package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/coocood/freecache"
	log "github.com/sirupsen/logrus"
)

// example API call
// http://api.openweathermap.org/data/2.5/weather?q=London,uk&APPID=0af09f7bce2fd9cbea44d6740f3c8e27

type WeatherApi struct {
	cache      *freecache.Cache
	citiesData map[string]*[]WeatherCity
}

func NewWeatherApi(cacheSizeMegabytes int, citiesDataPath string) *WeatherApi {
	megabyte := 1024 * 1024
	cacheSize := cacheSizeMegabytes * megabyte

	weatherApi := &WeatherApi{
		cache: freecache.NewCache(cacheSize),
	}

	loadedCities := 0
	citiesData, err := loadCitiesData(citiesDataPath)
	if err != nil {
		log.Errorf("failed to load weather cities data: %s", err)
	} else {
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
	}

	log.Debugf("loaded %d city names", len(weatherApi.citiesData))
	log.Debugf("total loaded cities: %d", loadedCities)

	return weatherApi
}

func (w *WeatherApi) GetWeatherCurrent(city WeatherCity, weatherApiKey string) (WeatherApiResponse, error) {
	weatherApiResponse := &WeatherApiResponse{}

	cacheKey := fmt.Sprintf("current::%d", city.ID)
	if currentCityWeatherBytes, err := w.cache.Get([]byte(cacheKey)); err == nil {
		log.Tracef("found current weather info for %s in cache", city.Name)
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
		log.Errorf("failed to write current weather cache for %s %d: %s", city.Name, city.ID, err)
	} else {
		log.Debugf("current weather cache set for city: %s", city.Name)
	}

	return *weatherApiResponse, nil
}

// returns something like sunny, cloudy, etc
func (w *WeatherApi) Get5DaysWeatherForecast(city WeatherCity, weatherApiKey string) ([]WeatherInfo, error) {
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

func (w *WeatherApi) getWeatherCity(geoInfo *GeoIpInfo) (WeatherCity, error) {
	cityName := strings.ToLower(geoInfo.City)
	citiesList, found := w.citiesData[cityName]
	if !found {
		return WeatherCity{}, ErrNotFound
	}

	if len(*citiesList) == 1 {
		return (*citiesList)[0], nil
	}

	country := strings.ToLower(geoInfo.CountryCode)
	for i := range *citiesList {
		c := (*citiesList)[i]
		if strings.ToLower(c.Country) == country {
			return c, nil
		}
	}

	return WeatherCity{}, ErrNotFound
}

func loadCitiesData(cityListDataPath string) ([]WeatherCity, error) {
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
