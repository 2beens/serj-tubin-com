package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeoIp_GetRequestGeoInfo(t *testing.T) {
	apiCallsCount := 0
	testServerHander := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallsCount++

		if r.Method == http.MethodGet && r.URL.Path == "/v2/info" &&
			r.URL.RawQuery == "apikey=dummy-api-key&ip=127.0.0.2" {
			WriteResponse(w, "application/json", `{
				"data": {
				  "timezone": {
					"id": "Australia/Sydney",
					"current_time": "2022-07-17T02:25:56+10:00",
					"code": "AEST",
					"is_daylight_saving": false,
					"gmt_offset": 36000
				  },
				  "ip": "127.0.0.2",
				  "type": "v4",
				  "connection": {
					"asn": 13335,
					"organization": "CLOUDFLARENET",
					"isp": "Cloudflare"
				  },
				  "location": {
					"geonames_id": 2147714,
					"latitude": -33.86714172363281,
					"longitude": 151.2071075439453,
					"zip": "2000",
					"continent": {
					  "code": "OC",
					  "name": "Oceania",
					  "name_translated": "Oceania"
					},
					"country": {
					  "alpha2": "AU",
					  "alpha3": "AUS",
					  "calling_codes": [
						"+61"
					  ],
					  "currencies": [
						{
						  "symbol": "AU$",
						  "name": "Australian Dollar",
						  "symbol_native": "$",
						  "decimal_digits": 2,
						  "rounding": 0,
						  "code": "AUD",
						  "name_plural": "Australian dollars"
						}
					  ],
					  "emoji": "🇦🇺",
					  "ioc": "AUS",
					  "languages": [
						{
						  "name": "English",
						  "name_native": "English"
						}
					  ],
					  "name": "Australia",
					  "name_translated": "Australia",
					  "timezones": [
						"Australia/Lord_Howe",
						"Antarctica/Macquarie",
						"Australia/Hobart",
						"Australia/Currie",
						"Australia/Melbourne",
						"Australia/Sydney",
						"Australia/Broken_Hill",
						"Australia/Brisbane",
						"Australia/Lindeman",
						"Australia/Adelaide",
						"Australia/Darwin",
						"Australia/Perth",
						"Australia/Eucla"
					  ],
					  "is_in_european_union": false
					},
					"city": {
					  "name": "Sydney",
					  "name_translated": "Sydney"
					},
					"region": {
					  "fips": "AS-02",
					  "alpha2": "AU-NSW",
					  "name": "New South Wales",
					  "name_translated": "New South Wales"
					}
				  }
				}
			  }`)
			return
		}

		http.Error(w, "unexpected path/method", http.StatusBadRequest)
	})
	testServer := httptest.NewServer(testServerHander)
	defer testServer.Close()

	geoIp := NewGeoIp(testServer.URL, "dummy-api-key", testServer.Client())
	require.NotNil(t, geoIp)

	req, err := http.NewRequest("GET", "/messages/count", nil)
	require.NoError(t, err)

	// will return geoIpInfo - development Berlin
	req.Header.Add("X-Real-Ip", "127.0.0.1:1234")
	geoIpInfo, err := geoIp.GetRequestGeoInfo(req)
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)
	assert.Equal(t, &devGeoIpInfo, geoIpInfo)

	// non-dev IP
	ipAddr := "127.0.0.2"
	req.Header.Set("X-Real-Ip", ipAddr)
	geoIpInfo, err = geoIp.GetRequestGeoInfo(req)
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)

	assert.Equal(t, "Sydney", geoIpInfo.Data.Location.City.Name)
	assert.Equal(t, "Australia", geoIpInfo.Data.Location.Country.Name)
	assert.Equal(t, "2000", geoIpInfo.Data.Location.Zip)
	assert.Equal(t, ipAddr, geoIpInfo.Data.IP)

	// again - has to be taken from the cache
	// TODO: maybe abstract the cache away and test it, like in the board_test.go
	geoIpInfo, err = geoIp.GetRequestGeoInfo(req)
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)

	assert.Equal(t, "Sydney", geoIpInfo.Data.Location.City.Name)
	assert.Equal(t, "Australia", geoIpInfo.Data.Location.Country.Name)
	assert.Equal(t, "2000", geoIpInfo.Data.Location.Zip)
	assert.Equal(t, ipAddr, geoIpInfo.Data.IP)
}

func TestGeoIp_ReadUserIP(t *testing.T) {
	geoIp := NewGeoIp("not-needed", "dummy", nil)
	require.NotNil(t, geoIp)

	req, err := http.NewRequest("-", "-", nil)
	require.NoError(t, err)

	// X-Real-Ip
	ip := "127.0.0.10"
	req.Header.Add("X-Real-Ip", ip)
	userIp, err := ReadUserIP(req)
	require.NoError(t, err)
	assert.Equal(t, ip, userIp)

	// X-Forwarded-For
	req, err = http.NewRequest("-", "-", nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-For", ip)
	userIp, err = ReadUserIP(req)
	require.NoError(t, err)
	assert.Equal(t, ip, userIp)

	// headers empty
	req, err = http.NewRequest("-", "-", nil)
	require.NoError(t, err)
	_, err = ReadUserIP(req)
	require.EqualError(t, err, "ip addr  is invalid")
}
