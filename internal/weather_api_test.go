package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeatherApi_NewWeatherApi(t *testing.T) {
	citiesData := getTestCitiesData()
	openWeatherTestKey := "open_weather_test_key"
	weatherApi := NewWeatherApi(openWeatherTestKey, citiesData)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 5)
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

	openWeatherTestKey := "open_weather_test_key"
	weatherApi := NewWeatherApi(openWeatherTestKey, citiesData)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 5)
}

func TestWeatherApi_GetWeatherCity(t *testing.T) {
	citiesData := getTestCitiesData()
	openWeatherTestKey := "open_weather_test_key"
	weatherApi := NewWeatherApi(openWeatherTestKey, citiesData)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 5)

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
	citiesData := getTestCitiesData()
	openWeatherTestKey := "open_weather_test_key"
	weatherApi := NewWeatherApi(openWeatherTestKey, citiesData)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 5)

	w, err := weatherApi.GetWeatherCurrent(2, "Berlin")
	require.NotNil(t, w)
	require.NoError(t, err)
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
	}
}
