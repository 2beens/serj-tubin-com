package pkg

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPIsLocal(t *testing.T) {
	cases := []struct {
		addr            string
		expectedIsLocal bool
	}{
		{addr: "83.12.53.65:2145", expectedIsLocal: false},
		{addr: "127.23.0.1:35325", expectedIsLocal: false},
		{addr: "172.20.0.1:60102", expectedIsLocal: true},
		{addr: "172.20.0.1:60096", expectedIsLocal: true},
		{addr: "172.200.0.1:60096", expectedIsLocal: true},
		{addr: "172.19.0.1:42452", expectedIsLocal: true},
		{addr: "172.0.0.1:42452", expectedIsLocal: true},
		{addr: "83.12.53.65:214", expectedIsLocal: false},
		{addr: "172.19.0.1:42452", expectedIsLocal: true},
		{addr: "172.0.0.1:352345", expectedIsLocal: true},
		{addr: "111.12.56.65:8080", expectedIsLocal: false},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.expectedIsLocal, IPIsLocal(tc.addr))
	}
}

func TestReadUserIP(t *testing.T) {
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
