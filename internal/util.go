package internal

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

func WriteResponse(w http.ResponseWriter, contentType, message string) {
	WriteResponseBytes(w, contentType, []byte(message))
}

func WriteResponseBytes(w http.ResponseWriter, contentType string, message []byte) {
	if contentType != "" {
		w.Header().Add("Content-Type", contentType)
	}

	if _, err := w.Write(message); err != nil {
		// TODO: add metrics and alarms instead... sometime in the future
		log.Errorf("failed to write response [%s]: %s", message, err)
	}
}

func LoadCitiesData(cityListDataPath string) ([]WeatherCity, error) {
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
