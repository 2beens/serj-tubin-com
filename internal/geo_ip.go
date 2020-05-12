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

//time="2020-05-12T17:45:34Z" level=error msg="error getting user ip: ip addr 127.0.0.1:34696 is invalid"
//time="2020-05-12T18:41:59Z" level=debug msg="calling geo ip info: https://freegeoip.app/json/94.134.177.115"
//time="2020-05-12T18:41:59Z" level=error msg="failed to unmarshal geo ip response bytes: json: cannot unmarshal number into Go struct field GeoIpResponse.latitude of type string"

func getRequestGeoInfo(r *http.Request) (GeoIpResponse, error) {
	userIp, err := readUserIP(r)
	if err != nil {
		return GeoIpResponse{}, fmt.Errorf("error getting user ip: %s", err.Error())
	}

	// allowed up to 15,000 queries per hour
	// https://freegeoip.app/
	geoIpUrl := fmt.Sprintf("https://freegeoip.app/json/%s", userIp)
	log.Debugf("calling geo ip info: %s", geoIpUrl)

	resp, err := http.Get(geoIpUrl)
	if err != nil {
		return GeoIpResponse{}, fmt.Errorf("error getting freegeoip response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return GeoIpResponse{}, fmt.Errorf("failed to read geo ip response bytes: %s", err)
	}

	geoIpResponse := &GeoIpResponse{}
	err = json.Unmarshal(respBytes, geoIpResponse)
	if err != nil {
		return GeoIpResponse{}, fmt.Errorf("failed to unmarshal geo ip response bytes: %s", err)
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
