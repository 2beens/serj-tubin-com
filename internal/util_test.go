package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPassword(t *testing.T) {
	passwordHash, err := HashPassword("sr")
	require.NoError(t, err)
	assert.NotEmpty(t, passwordHash)

	assert.True(t, CheckPasswordHash("sr", "$2a$14$z8cd4yJpzP40Qh2F2BhiMO.sOm4YAIaf30pmUKLOaISojD9HnXgaG"))
	assert.True(t, CheckPasswordHash("sr", passwordHash))
}

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
