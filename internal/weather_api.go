package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

// example API call
// http://api.openweathermap.org/data/2.5/weather?q=London,uk&APPID=0af09f7bce2fd9cbea44d6740f3c8e27

// TODO: cache responses

func getWeatherCurrent(geoInfo *GeoIpInfo, weatherApiKey string) (WeatherApiResponse, error) {
	weatherApiUrl := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?q=%s,%s&APPID=%s", geoInfo.City, geoInfo.CountryCode, weatherApiKey)
	log.Debugf("calling weather api info: %s", weatherApiUrl)

	resp, err := http.Get(weatherApiUrl)
	if err != nil {
		return WeatherApiResponse{}, fmt.Errorf("error getting weather api response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return WeatherApiResponse{}, fmt.Errorf("failed to read weather api response bytes: %s", err)
	}

	weatherApiResponse := &WeatherApiResponse{}
	err = json.Unmarshal(respBytes, weatherApiResponse)
	if err != nil {
		return WeatherApiResponse{}, fmt.Errorf("failed to unmarshal weather api response bytes: %s", err)
	}

	return *weatherApiResponse, nil
}

// returns something like sunny, cloudy, etc
func getWeatherTomorrow(city WeatherCity, weatherApiKey string) ([]string, error) {
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
