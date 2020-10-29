package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/coocood/freecache"
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

type GeoIp struct {
	cache *freecache.Cache
	mutex sync.RWMutex
}

func NewGeoIp() *GeoIp {
	megabyte := 1024 * 1024
	cacheSize := 50 * megabyte

	return &GeoIp{
		cache: freecache.NewCache(cacheSize),
	}
}

func (gi *GeoIp) GetRequestGeoInfo(r *http.Request) (*GeoIpInfo, error) {
	userIp, err := gi.ReadUserIP(r)
	if err != nil {
		return nil, fmt.Errorf("error getting user ip: %s", err.Error())
	}

	// TODO: control this with config and environment
	// used for development
	if userIp == "127.0.0.1" {
		log.Debugf("request geo info: returning development 127.0.0.1 / Berlin")
		return &GeoIpInfo{
			Ip:          "127.0.0.1",
			CountryCode: "de",
			CountryName: "Germany",
			RegionCode:  "",
			RegionName:  "",
			City:        "Berlin",
			ZipCode:     "12099",
			TimeZone:    "",
			Latitude:    0,
			Longitude:   0,
			MetroCode:   0,
		}, nil
	}

	geoIpResponse := &GeoIpInfo{}

	// TODO: seems like freecache already solves sync issues (can be removed?)
	gi.mutex.RLock()
	if geoIpInfoBytes, err := gi.cache.Get([]byte(userIp)); err == nil {
		log.Tracef("found geo ip info for %s in cache", userIp)
		if err = json.Unmarshal(geoIpInfoBytes, geoIpResponse); err == nil {
			gi.mutex.RUnlock()
			return geoIpResponse, nil
		}

		log.Errorf("failed to unmarshal cached geo ip info %s: %s", userIp, err)
		// continue, and try getting it from Geo IP API
	} else {
		log.Debugf("failed to get cached geo ip info value for %s: %s", userIp, err)
	}
	gi.mutex.RUnlock()

	// allowed up to 15,000 queries per hour
	// https://freegeoip.app/
	geoIpUrl := fmt.Sprintf("https://freegeoip.app/json/%s", userIp)
	log.Debugf("calling geo ip info: %s", geoIpUrl)

	resp, err := http.Get(geoIpUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting freegeoip response: %s", err.Error())
	}

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read geo ip response bytes: %s", err)
	}

	err = json.Unmarshal(respBytes, geoIpResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal geo ip response bytes: %s", err)
	}

	// set cache
	gi.mutex.Lock()
	if err = gi.cache.Set([]byte(userIp), respBytes, GeoIpCacheExpire); err != nil {
		log.Errorf("failed to write geo ip cache for %s: %s", userIp, err)
	} else {
		log.Debugf("geo ip cache set for: %s", userIp)
	}
	gi.mutex.Unlock()

	return geoIpResponse, nil
}

func (gi *GeoIp) ReadUserIP(r *http.Request) (string, error) {
	ipAddr := r.Header.Get("X-Real-Ip")
	if ipAddr == "" {
		ipAddr = r.Header.Get("X-Forwarded-For")
	}
	if ipAddr == "" {
		ipAddr = r.RemoteAddr
	}

	// TODO: control this with config and environment
	// used for development
	if strings.HasPrefix(ipAddr, "127.0.0.1") {
		log.Debugf("read user IP: returning development 127.0.0.1 / Berlin")
		return "127.0.0.1", nil
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
