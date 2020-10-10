package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewWeatherApi(t *testing.T) {
	citiesData := getTestCitiesData()
	weatherApi := NewWeatherApi(citiesData)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 3)
}

func Test_NewWeatherApi_DuplicateCities(t *testing.T) {
	citiesData := getTestCitiesData()
	// add city 0 twice, make sure all ok
	citiesData = append(citiesData, WeatherCity{
		ID:       1,
		Name:     "city1",
		State:    "state1",
		Country:  "RS",
		Coord:    Coordinate{},
		Timezone: 0,
		Sunrise:  0,
		Sunset:   0,
	})

	weatherApi := NewWeatherApi(citiesData)
	assert.NotNil(t, weatherApi)
	assert.Len(t, weatherApi.citiesData, 3)
}

func getTestCitiesData() []WeatherCity {
	return []WeatherCity{
		{
			ID:       0,
			Name:     "city0",
			State:    "state0",
			Country:  "RS",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:       1,
			Name:     "city1",
			State:    "state1",
			Country:  "RS",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
		{
			ID:       2,
			Name:     "city2",
			State:    "state2",
			Country:  "RS",
			Coord:    Coordinate{},
			Timezone: 0,
			Sunrise:  0,
			Sunset:   0,
		},
	}
}
