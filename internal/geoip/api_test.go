package geoip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/2beens/serjtubincom/pkg"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ipInfoTestResponse = `{
	  "ip": "127.0.0.2",
	  "hostname": "153.red-80-36-233.staticip.rima-tde.net",
	  "city": "Palma",
	  "region": "Balearic Islands",
	  "country": "ES",
	  "loc": "39.5680,2.6835",
	  "org": "AS3352 TELEFONICA DE ESPANA S.A.U.",
	  "postal": "07198",
	  "timezone": "Europe/Madrid"
	}`
)

func TestGeoIp_GetRequestGeoInfo(t *testing.T) {
	apiCallsCount := 0
	testServerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallsCount++

		if r.Method == http.MethodGet && r.URL.Path == "/127.0.0.2" {
			pkg.WriteResponse(w, "application/json", ipInfoTestResponse)
			return
		}

		http.Error(w, "unexpected path/method", http.StatusBadRequest)
	})
	testServer := httptest.NewServer(testServerHandler)
	defer testServer.Close()

	db, mock := redismock.NewClientMock()
	mock.ExpectGet("ip-info::127.0.0.1").SetVal("")

	geoIp := NewApi("dummyapikey", testServer.Client(), db)
	require.NotNil(t, geoIp)
	testServerURL, err := url.Parse(testServer.URL)
	require.NoError(t, err)
	geoIp.ipInfoClient.BaseURL = testServerURL

	ctx := context.Background()
	// will return geoIpInfo - development Berlin
	geoIpInfo, err := geoIp.GetIPGeoInfo(ctx, "localhost")
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)
	assert.Equal(t, devGeoIpInfo, geoIpInfo)

	// non-dev IP
	geoIpInfo, err = geoIp.GetIPGeoInfo(ctx, "127.0.0.2")
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)

	assert.Equal(t, "Palma", geoIpInfo.City)
	assert.Equal(t, "ES", geoIpInfo.Country)
	assert.Equal(t, "07198", geoIpInfo.Postal)
	assert.Equal(t, "127.0.0.2", geoIpInfo.IP.String())

	// TODO: test the case when getting the value from redis cache
}
