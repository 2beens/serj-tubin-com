package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Weather struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type WeatherMain struct {
	Temp      float64 `json:"temp"`
	FeelsLike float64 `json:"feels_like"`
	TempMin   float64 `json:"temp_min"`
	TempMax   float64 `json:"temp_max"`
	Pressure  int     `json:"pressure"`
	Humidity  int     `json:"humidity"`
}

type Coordinate struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

type Wind struct {
	Speed float64 `json:"speed"`
	Deg   int     `json:"deg"`
}

type WeatherApiResponse struct {
	Coord      Coordinate  `json:"coord"`
	Weather    []Weather   `json:"weather"`
	Base       string      `json:"base"`
	Main       WeatherMain `json:"main"`
	Visibility int         `json:"visibility"`
	Wind       Wind        `json:"wind"`
	Clouds     struct {
		All int `json:"all"`
	} `json:"clouds"`
	Dt  int `json:"dt"`
	Sys struct {
		Type    int    `json:"type"`
		ID      int    `json:"id"`
		Country string `json:"country"`
		Sunrise int    `json:"sunrise"`
		Sunset  int    `json:"sunset"`
	} `json:"sys"`
	Timezone int    `json:"timezone"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Cod      int    `json:"cod"`
}

// example API call
// http://api.openweathermap.org/data/2.5/weather?q=London,uk&APPID=0af09f7bce2fd9cbea44d6740f3c8e27

func getWeatherInfo(geoInfo GeoIpInfo, weatherApiKey string) (WeatherApiResponse, error) {
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
