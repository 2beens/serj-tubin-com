package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type GeoIpInfo struct {
	Ip          string  `json:"ip"`
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	RegionCode  string  `json:"region_code"`
	RegionName  string  `json:"region_name"`
	City        string  `json:"city"`
	ZipCode     string  `json:"zip_code"`
	TimeZone    string  `json:"time_zone"`
	Latitude    float32 `json:"latitude"`
	Longitude   float32 `json:"longitude"`
	MetroCode   int     `json:"metro_code"`
}

// TODO: cache geo ip info

func getRequestGeoInfo(r *http.Request) (GeoIpInfo, error) {
	userIp, err := readUserIP(r)
	if err != nil {
		return GeoIpInfo{}, fmt.Errorf("error getting user ip: %s", err.Error())
	}

	// allowed up to 15,000 queries per hour
	// https://freegeoip.app/
	geoIpUrl := fmt.Sprintf("https://freegeoip.app/json/%s", userIp)
	log.Debugf("calling geo ip info: %s", geoIpUrl)

	resp, err := http.Get(geoIpUrl)
	if err != nil {
		return GeoIpInfo{}, fmt.Errorf("error getting freegeoip response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GeoIpInfo{}, fmt.Errorf("failed to read geo ip response bytes: %s", err)
	}

	geoIpResponse := &GeoIpInfo{}
	err = json.Unmarshal(respBytes, geoIpResponse)
	if err != nil {
		return GeoIpInfo{}, fmt.Errorf("failed to unmarshal geo ip response bytes: %s", err)
	}

	return *geoIpResponse, nil
}

func readUserIP(r *http.Request) (string, error) {
	ipAddr := r.Header.Get("X-Real-Ip")
	if ipAddr == "" {
		ipAddr = r.Header.Get("X-Forwarded-For")
	}
	if ipAddr == "" {
		ipAddr = r.RemoteAddr
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
