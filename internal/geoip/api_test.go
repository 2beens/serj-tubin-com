package geoip

//import (
//	"context"
//	"net/http"
//	"net/http/httptest"
//	"testing"
//
//	"github.com/2beens/serjtubincom/pkg"
//
//	"github.com/go-redis/redismock/v8"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//)
//
//const (
//	ipInfoTestResponse = `{
//	  "ip": "127.0.0.2",
//	  "hostname": "153.red-80-36-233.staticip.rima-tde.net",
//	  "city": "Palma",
//	  "region": "Balearic Islands",
//	  "country": "ES",
//	  "loc": "39.5680,2.6835",
//	  "org": "AS3352 TELEFONICA DE ESPANA S.A.U.",
//	  "postal": "07198",
//	  "timezone": "Europe/Madrid"
//	}`
//)
//
//func TestGeoIp_GetRequestGeoInfo(t *testing.T) {
//	apiCallsCount := 0
//	testServerHander := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		apiCallsCount++
//
//		if r.Method == http.MethodGet && r.URL.Path == "/v2/info" &&
//			r.URL.RawQuery == "apikey=dummy-api-key&ip=127.0.0.2" {
//			pkg.WriteResponse(w, "application/json", ipInfoTestResponse)
//			return
//		}
//
//		http.Error(w, "unexpected path/method", http.StatusBadRequest)
//	})
//	testServer := httptest.NewServer(testServerHander)
//	defer testServer.Close()
//
//	db, mock := redismock.NewClientMock()
//	mock.ExpectGet("ip-info::127.0.0.1").SetVal("")
//
//	geoIp := NewApi(testServer.URL, "dummy-api-key", testServer.Client(), db)
//	require.NotNil(t, geoIp)
//
//	ctx := context.Background()
//	// will return geoIpInfo - development Berlin
//	geoIpInfo, err := geoIp.GetIPGeoInfo(ctx, "localhost")
//	require.NoError(t, err)
//	require.NotNil(t, geoIpInfo)
//	assert.Equal(t, &devGeoIpInfo, geoIpInfo)
//
//	// non-dev IP
//	geoIpInfo, err = geoIp.GetIPGeoInfo(ctx, "127.0.0.2")
//	require.NoError(t, err)
//	require.NotNil(t, geoIpInfo)
//
//	assert.Equal(t, "Palma", geoIpInfo.City)
//	assert.Equal(t, "ES", geoIpInfo.Country)
//	assert.Equal(t, "07198", geoIpInfo.Postal)
//	assert.Equal(t, "127.0.0.2", geoIpInfo.IP)
//
//	// TODO: test the case when getting the value from redis cache
//}
//
//func TestGeoIp_ReadUserIP(t *testing.T) {
//	db, mock := redismock.NewClientMock()
//	mock.ExpectGet("ip-info::127.0.0.1").SetVal("")
//
//	geoIp := NewApi("not-needed", "dummy", nil, db)
//	require.NotNil(t, geoIp)
//
//	req, err := http.NewRequest("-", "-", nil)
//	require.NoError(t, err)
//
//	// X-Real-Ip
//	ip := "127.0.0.10"
//	req.Header.Add("X-Real-Ip", ip)
//	userIp, err := pkg.ReadUserIP(req)
//	require.NoError(t, err)
//	assert.Equal(t, ip, userIp)
//
//	// X-Forwarded-For
//	req, err = http.NewRequest("-", "-", nil)
//	require.NoError(t, err)
//	req.Header.Set("X-Forwarded-For", ip)
//	userIp, err = pkg.ReadUserIP(req)
//	require.NoError(t, err)
//	assert.Equal(t, ip, userIp)
//
//	// headers empty
//	req, err = http.NewRequest("-", "-", nil)
//	require.NoError(t, err)
//	_, err = pkg.ReadUserIP(req)
//	require.EqualError(t, err, "ip addr  is invalid")
//}
