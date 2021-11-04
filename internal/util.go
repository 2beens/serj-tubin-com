package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/2beens/serjtubincom/pkg"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var (
	localDockerIpRegex = regexp.MustCompile(`^172\.\d{1,3}\.0\.1:\d{1,5}`)
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
	return pkg.BytesToString(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func IPIsLocal(ipAddr string) bool {
	// used in local development ?
	if strings.HasPrefix(ipAddr, "127.0.0.1:") {
		return true
	}

	// user within docker container ?
	return localDockerIpRegex.MatchString(ipAddr)
}

func ReadUserIP(r *http.Request) (string, error) {
	ipAddr := r.Header.Get("X-Real-Ip")
	if ipAddr == "" {
		ipAddr = r.Header.Get("X-Forwarded-For")
	}
	if ipAddr == "" {
		ipAddr = r.RemoteAddr
	}

	// used in development
	if IPIsLocal(ipAddr) {
		log.Debugf("read user IP: returning development localhost / Berlin")
		return "localhost", nil
	}

	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return "", fmt.Errorf("ip addr %s is invalid", ipAddr)
	}

	if strings.Contains(ipAddr, ":") {
		ipAddr = strings.Split(ipAddr, ":")[0]
	}

	return ipAddr, nil
}
