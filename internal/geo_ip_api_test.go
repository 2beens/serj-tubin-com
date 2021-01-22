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

		if r.Method == http.MethodGet && r.URL.Path == "/json/127.0.0.2" {
			WriteResponse(w, "application/json", `{
				"ip":"127.0.0.2",
				"country_code":"RS",
				"country_name":"Serbia",
				"region_code":"V",
				"region_name":"Vojvodina",
				"city":"Novi Sad",
				"zip_code":"21000",
				"time_zone":"UTC+2",
				"latitude":10,
				"longitude":20,
				"metro_code":30
			}`)
			return
		}

		http.Error(w, "unexpected path/method", http.StatusBadRequest)
	})
	testServer := httptest.NewServer(testServerHander)
	defer testServer.Close()

	geoIp := NewGeoIp(testServer.URL, testServer.Client())
	require.NotNil(t, geoIp)

	req, err := http.NewRequest("GET", "/messages/count", nil)
	require.NoError(t, err)

	// will return geoIpInfo - development Berlin
	req.Header.Add("X-Real-Ip", "127.0.0.1")
	geoIpInfo, err := geoIp.GetRequestGeoInfo(req)
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)
	assert.Equal(t, &devGeoIpInfo, geoIpInfo)

	// non-dev IP
	req.Header.Set("X-Real-Ip", "127.0.0.2")
	geoIpInfo, err = geoIp.GetRequestGeoInfo(req)
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)

	assert.Equal(t, "Novi Sad", geoIpInfo.City)
	assert.Equal(t, "Serbia", geoIpInfo.CountryName)
	assert.Equal(t, "21000", geoIpInfo.ZipCode)
	assert.Equal(t, "127.0.0.2", geoIpInfo.Ip)

	// again - has to be taken from the cache
	// TODO: maybe abstract the cache away and test it, like in the board_test.go
	geoIpInfo, err = geoIp.GetRequestGeoInfo(req)
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)

	assert.Equal(t, "Novi Sad", geoIpInfo.City)
	assert.Equal(t, "Serbia", geoIpInfo.CountryName)
	assert.Equal(t, "21000", geoIpInfo.ZipCode)
	assert.Equal(t, "127.0.0.2", geoIpInfo.Ip)
}

func TestGeoIp_ReadUserIP(t *testing.T) {
	geoIp := NewGeoIp("not-needed", nil)
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
