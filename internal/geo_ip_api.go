package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/coocood/freecache"
	log "github.com/sirupsen/logrus"
)

// type GeoIpInfo struct {
// 	Ip          string  `json:"ip"`
// 	CountryCode string  `json:"country_code"`
// 	CountryName string  `json:"country_name"`
// 	RegionCode  string  `json:"region_code"`
// 	RegionName  string  `json:"region_name"`
// 	City        string  `json:"city"`
// 	ZipCode     string  `json:"zip_code"`
// 	TimeZone    string  `json:"time_zone"`
// 	Latitude    float32 `json:"latitude"`
// 	Longitude   float32 `json:"longitude"`
// 	MetroCode   int     `json:"metro_code"`
// }

type GeoIp struct {
	ipBaseEndpoint string
	ipBaseAPIKey   string
	httpClient     *http.Client
	cache          *freecache.Cache
	mutex          sync.RWMutex
}

var (
	// devGeoIpInfo = GeoIpInfo{
	// 	Ip:          "127.0.0.1",
	// 	CountryCode: "de",
	// 	CountryName: "Germany",
	// 	RegionCode:  "",
	// 	RegionName:  "",
	// 	City:        "Berlin",
	// 	ZipCode:     "12099",
	// 	TimeZone:    "",
	// 	Latitude:    0,
	// 	Longitude:   0,
	// 	MetroCode:   0,
	// }

	devGeoIpInfo = GeoIpInfo{
		Data: GeoIpInfoData{
			IP: "127.0.0.1",
			Location: GeoLocation{
				City: City{
					Name: "Berlin",
				},
				Country: Country{
					Alpha2: "DE",
					Alpha3: "DEU",
					Name:   "Germany",
				},
			},
		},
	}
)

func NewGeoIp(ipBaseEndpoint, ipBaseAPIKey string, httpClient *http.Client) *GeoIp {
	megabyte := 1024 * 1024
	cacheSize := 50 * megabyte

	return &GeoIp{
		ipBaseEndpoint: ipBaseEndpoint,
		ipBaseAPIKey:   ipBaseAPIKey,
		httpClient:     httpClient,
		cache:          freecache.NewCache(cacheSize),
	}
}

func (gi *GeoIp) GetRequestGeoInfo(r *http.Request) (*GeoIpInfo, error) {
	userIp, err := ReadUserIP(r)
	if err != nil {
		return nil, fmt.Errorf("error getting user ip: %s", err.Error())
	}

	// used for development
	if userIp == "localhost" {
		log.Debugf("request geo info: returning development localhost / Berlin")
		return &devGeoIpInfo, nil
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

	geoIpUrl := fmt.Sprintf("%s/v2/info?apikey=%s&ip=%s", gi.ipBaseEndpoint, gi.ipBaseAPIKey, userIp)
	log.Debugf("calling geo ip info: %s", geoIpUrl)

	resp, err := gi.httpClient.Get(geoIpUrl)
	if err != nil {
		return nil, fmt.Errorf("error getting freegeoip response: %s", err.Error())
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read geo ip response bytes: %s", err)
	}

	log.Debugf("calling geo ip info for ip: %s, response: %s", geoIpUrl, respBytes)

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

type GeoIpInfo struct {
	Data GeoIpInfoData `json:"data"`
}

type GeoIpInfoData struct {
	Timezone struct {
		ID               string    `json:"id"`
		CurrentTime      time.Time `json:"current_time"`
		Code             string    `json:"code"`
		IsDaylightSaving bool      `json:"is_daylight_saving"`
		GmtOffset        int       `json:"gmt_offset"`
	} `json:"timezone"`
	IP         string `json:"ip"`
	Type       string `json:"type"`
	Connection struct {
		Asn          int    `json:"asn"`
		Organization string `json:"organization"`
		Isp          string `json:"isp"`
	} `json:"connection"`
	Location GeoLocation `json:"location"`
}

type GeoLocation struct {
	GeonamesID int     `json:"geonames_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Zip        string  `json:"zip"`
	Continent  struct {
		Code           string `json:"code"`
		Name           string `json:"name"`
		NameTranslated string `json:"name_translated"`
	} `json:"continent"`
	Country Country `json:"country"`
	City    City    `json:"city"`
	Region  Region  `json:"region"`
}

type Region struct {
	Fips           string `json:"fips"`
	Alpha2         string `json:"alpha2"`
	Name           string `json:"name"`
	NameTranslated string `json:"name_translated"`
}

type City struct {
	Name           string `json:"name"`
	NameTranslated string `json:"name_translated"`
}

type Country struct {
	Alpha2       string     `json:"alpha2"`
	Alpha3       string     `json:"alpha3"`
	CallingCodes []string   `json:"calling_codes"`
	Currencies   []Currency `json:"currencies"`
	Emoji        string     `json:"emoji"`
	Ioc          string     `json:"ioc"`
	Languages    []struct {
		Name       string `json:"name"`
		NameNative string `json:"name_native"`
	} `json:"languages"`
	Name              string   `json:"name"`
	NameTranslated    string   `json:"name_translated"`
	Timezones         []string `json:"timezones"`
	IsInEuropeanUnion bool     `json:"is_in_european_union"`
}

type Currency struct {
	Symbol        string `json:"symbol"`
	Name          string `json:"name"`
	SymbolNative  string `json:"symbol_native"`
	DecimalDigits int    `json:"decimal_digits"`
	Rounding      int    `json:"rounding"`
	Code          string `json:"code"`
	NamePlural    string `json:"name_plural"`
}
