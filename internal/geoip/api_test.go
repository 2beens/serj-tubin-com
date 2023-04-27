package geoip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/2beens/serjtubincom/pkg"

	"github.com/go-redis/redismock/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
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
	expectedCacheSetJson = `{"ip":"127.0.0.2","hostname":"153.red-80-36-233.staticip.rima-tde.net","city":"Palma","region":"Balearic Islands","country":"ES","country_name":"Spain","country_flag":{"emoji":"🇪🇸","unicode":"U+1F1EA U+1F1F8"},"country_currency":{"code":"EUR","symbol":"€"},"continent":{"code":"EU","name":"Europe"},"isEU":true,"loc":"39.5680,2.6835","org":"AS3352 TELEFONICA DE ESPANA S.A.U.","postal":"07198","timezone":"Europe/Madrid"}`
)

// TestMain will run goleak after all tests have been run in the package
// to detect any goroutine leaks
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// INFO: https://github.com/go-redis/redis/issues/1029
		goleak.IgnoreTopFunction(
			"github.com/go-redis/redis/v8/internal/pool.(*ConnPool).reaper",
		),
	)
}

func TestGeoIp_GetRequestGeoInfo(t *testing.T) {
	apiCallsCount := 0
	testServerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallsCount++

		if r.Method == http.MethodGet && r.URL.Path == "/127.0.0.2" {
			pkg.WriteJSONResponseOK(w, ipInfoTestResponse)
			return
		}

		http.Error(w, "unexpected path/method", http.StatusBadRequest)
	})
	testServer := httptest.NewServer(testServerHandler)
	defer testServer.Close()

	db, mock := redismock.NewClientMock()

	geoIp := NewApi(testServer.URL, "dummyapikey", testServer.Client(), db)
	require.NotNil(t, geoIp)

	ctx := context.Background()
	// will return geoIpInfo - development Berlin
	geoIpInfo, err := geoIp.GetIPGeoInfo(ctx, "localhost")
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)
	assert.Equal(t, devGeoIpInfo, geoIpInfo)

	// non-dev IP
	mock.ExpectGet("ip-info::127.0.0.2").SetVal("")
	mock.ExpectSet("ip-info::127.0.0.2", []byte(expectedCacheSetJson), 0).RedisNil()
	geoIpInfo, err = geoIp.GetIPGeoInfo(ctx, "127.0.0.2")
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)

	assert.Equal(t, "Palma", geoIpInfo.City)
	assert.Equal(t, "ES", geoIpInfo.Country)
	assert.Equal(t, "07198", geoIpInfo.Postal)
	assert.Equal(t, "127.0.0.2", geoIpInfo.IP.String())

	// test the case when getting the value from redis cache
	mock.ExpectGet("ip-info::127.0.0.2").SetVal(ipInfoTestResponse)
	geoIpInfo, err = geoIp.GetIPGeoInfo(ctx, "127.0.0.2")
	require.NoError(t, err)
	require.NotNil(t, geoIpInfo)

	assert.Equal(t, "Palma", geoIpInfo.City)
	assert.Equal(t, "ES", geoIpInfo.Country)
	assert.Equal(t, "07198", geoIpInfo.Postal)
	assert.Equal(t, "127.0.0.2", geoIpInfo.IP.String())

	// first call returned local dev info, second call was made, and last call found data in cache
	// which makes it to only 1 API call
	assert.Equal(t, 1, apiCallsCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}
