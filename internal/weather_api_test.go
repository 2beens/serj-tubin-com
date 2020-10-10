package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeatherApi_NewWeatherApi(t *testing.T) {
	citiesData := getTestCitiesData()
	weatherApi := NewWeatherApi("http://test.owa", "open_weather_test_key", citiesData, nil)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 6)
}

func TestWeatherApi_NewWeatherApi_DuplicateCities(t *testing.T) {
	citiesData := getTestCitiesData()
	// add city 0 twice, make sure all ok
	citiesData = append(citiesData, WeatherCity{
		ID:       1,
		Name:     "Virovitica",
		State:    "Medjumurje",
		Country:  "HR",
		Coord:    Coordinate{},
		Timezone: 0,
		Sunrise:  0,
		Sunset:   0,
	})

	weatherApi := NewWeatherApi("http://test.owa", "open_weather_test_key", citiesData, nil)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 6)
}

func TestWeatherApi_GetWeatherCity(t *testing.T) {
	citiesData := getTestCitiesData()
	weatherApi := NewWeatherApi("http://test.owa", "open_weather_test_key", citiesData, nil)
	assert.NotNil(t, weatherApi)

	// not existent city
	c, err := weatherApi.GetWeatherCity("blabla", "RS")
	assert.Nil(t, c)
	assert.Equal(t, ErrNotFound, err)

	// existent city
	c, err = weatherApi.GetWeatherCity("Virovitica", "HR")
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, 1, c.ID)
	c, err = weatherApi.GetWeatherCity("virovitica", "hr")
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, 1, c.ID)

	// wrong country code
	c, err = weatherApi.GetWeatherCity("Novi Grad", "GR")
	assert.Nil(t, c)
	assert.Equal(t, ErrNotFound, err)
}

func TestWeatherApi_GetWeatherCurrent(t *testing.T) {
	londonCityId := 2643743

	// there should be only 1 api call, since the second time we call for
	// current weather, it's retrieved from the cache
	apiCallsCount := 0

	testServerHander := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallsCount++
		assert.Equal(t, fmt.Sprintf("/?id=%d&appid=open_weather_test_key", londonCityId), r.RequestURI)
		assert.Equal(t, http.MethodGet, r.Method)
		w.Write([]byte(weatherApiTestResponses[londonCityId]))
	})
	testServer := httptest.NewServer(testServerHander)
	defer testServer.Close()

	citiesData := getTestCitiesData()
	openWeatherTestKey := "open_weather_test_key"
	weatherApi := NewWeatherApi(testServer.URL, openWeatherTestKey, citiesData, testServer.Client())
	assert.NotNil(t, weatherApi)

	// with cache miss
	weather, err := weatherApi.GetWeatherCurrent(londonCityId, "London")
	require.NotNil(t, weather)
	require.NoError(t, err)
	assert.Equal(t, "London", weather.Name)
	assert.Equal(t, londonCityId, weather.ID)

	require.Len(t, weather.WeatherDescriptions, 1)
	assert.Equal(t, 300, weather.WeatherDescriptions[0].ID)
	assert.Equal(t, "light intensity drizzle", weather.WeatherDescriptions[0].Description)
	assert.Equal(t, "Drizzle", weather.WeatherDescriptions[0].Main)
	assert.Equal(t, "09d", weather.WeatherDescriptions[0].Icon)

	// with cache hit
	weather, err = weatherApi.GetWeatherCurrent(londonCityId, "London")
	require.NotNil(t, weather)
	require.NoError(t, err)
	assert.Equal(t, "London", weather.Name)
	assert.Equal(t, londonCityId, weather.ID)

	require.Len(t, weather.WeatherDescriptions, 1)
	assert.Equal(t, 300, weather.WeatherDescriptions[0].ID)
	assert.Equal(t, "light intensity drizzle", weather.WeatherDescriptions[0].Description)
	assert.Equal(t, "Drizzle", weather.WeatherDescriptions[0].Main)
	assert.Equal(t, "09d", weather.WeatherDescriptions[0].Icon)

	// second time we request - cache should be hit
	assert.Equal(t, 1, apiCallsCount)
}

func getTestCitiesData() []WeatherCity {
	return []WeatherCity{
		{
			ID:       0,
			Name:     "Novi Sad",
			State:    "Vojvodina",
			Country:  "RS",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:       1,
			Name:     "Virovitica",
			State:    "Medjumurje",
			Country:  "HR",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:       2,
			Name:     "Berlin",
			State:    "Berlin",
			Country:  "DE",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:       3,
			Name:     "Szolnok",
			State:    "Rendorseg",
			Country:  "HU",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:       4,
			Name:     "Novi Grad",
			State:    "Banjalucka",
			Country:  "BH",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:       5,
			Name:     "Novi Grad",
			State:    "Dalmacija",
			Country:  "HR",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:      2643743,
			Name:    "London",
			State:   "England",
			Country: "GB",
			Coord: Coordinate{
				Lon: -0.13,
				Lat: 51.51,
			},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
	}
}

var (
	weatherApiTestResponses = map[int]string{
		2643743: `
{
 "coord": {
   "lon": -0.13,
   "lat": 51.51
 },
 "weather": [
   {
	 "id": 300,
	 "main": "Drizzle",
	 "description": "light intensity drizzle",
	 "icon": "09d"
   }
 ],
 "base": "stations",
 "main": {
   "temp": 280.32,
   "pressure": 1012,
   "humidity": 81,
   "temp_min": 279.15,
   "temp_max": 281.15
 },
 "visibility": 10000,
 "wind": {
   "speed": 4.1,
   "deg": 80
 },
 "clouds": {
   "all": 90
 },
 "dt": 1485789600,
 "sys": {
   "type": 1,
   "id": 5091,
   "message": 0.0103,
   "country": "GB",
   "sunrise": 1485762037,
   "sunset": 1485794875
 },
 "id": 2643743,
 "name": "London",
 "cod": 200
}`,
	}
)
