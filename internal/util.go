package internal

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
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

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}
