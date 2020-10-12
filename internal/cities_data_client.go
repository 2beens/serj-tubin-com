package internal

import (
	"fmt"

	as "github.com/aerospike/aerospike-client-go"
)

type CitiesDataClient struct {
	aeroClient *as.Client
	// TODO: add cache
}

func NewCitiesDataClient(aeroHost string, aeroPort int, namespace string, rawCitiesData []WeatherCity) (*CitiesDataClient, error) {
	client, err := as.NewClient(aeroHost, aeroPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create aero client: %w", err)
	}

	cdClient := &CitiesDataClient{
		aeroClient: client,
	}

	cdClient.setupCitiesData(rawCitiesData)

	return cdClient, nil
}

func (c *CitiesDataClient) setupCitiesData(rawCitiesData []WeatherCity) (loadedCities int) {
	//citiesData := make(map[string]*[]WeatherCity)
	//for i := range rawCitiesData {
	//	loadedCities++
	//	city := rawCitiesData[i]
	//	cityName := strings.ToLower(city.Name)
	//	if cList, ok := weatherApi.citiesData[cityName]; ok {
	//		*cList = append(*cList, city)
	//	} else {
	//		weatherApi.citiesData[cityName] = &[]WeatherCity{city}
	//	}
	//}
	//
	//log.Debugf("loaded %d city names", len(weatherApi.citiesData))
	//log.Debugf("total loaded cities: %d", loadedCities)

	// TODO: not sure this is a good idea

	return 0
}

// TODO: check if needed
func (c *CitiesDataClient) Close() {
	if c != nil && c.aeroClient != nil {
		c.aeroClient.Close()
	}
}
