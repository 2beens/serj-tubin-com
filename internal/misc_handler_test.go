package internal

import (
	"net/http"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMiscHandler(t *testing.T) {
	mainRouter := mux.NewRouter()
	handler := NewMiscHandler(mainRouter, nil, nil, "dummy", &LoginSession{})
	require.NotNil(t, handler)
	require.NotNil(t, mainRouter)

	for caseName, route := range map[string]struct {
		name   string
		path   string
		method string
	}{
		"route-get": {
			name:   "root",
			path:   "/",
			method: "GET",
		},
		"route-post": {
			name:   "root",
			path:   "/",
			method: "POST",
		},
		"route-options": {
			name:   "root",
			path:   "/",
			method: "OPTIONS",
		},
		"quote": {
			name:   "quote",
			path:   "/quote/random",
			method: "GET",
		},
		"whereami": {
			name:   "whereami",
			path:   "/whereami",
			method: "GET",
		},
		"myip": {
			name:   "myip",
			path:   "/myip",
			method: "GET",
		},
		"version": {
			name:   "version",
			path:   "/version",
			method: "GET",
		},
		"login": {
			name:   "login",
			path:   "/login",
			method: "POST",
		},
		"logout": {
			name:   "logout",
			path:   "/logout",
			method: "GET",
		},
	} {
		t.Run(caseName, func(t *testing.T) {
			req, err := http.NewRequest(route.method, route.path, nil)
			require.NoError(t, err)

			routeMatch := &mux.RouteMatch{}
			route := mainRouter.Get(route.name)
			require.NotNil(t, route)
			isMatch := route.Match(req, routeMatch)
			assert.True(t, isMatch, caseName)
		})
	}
}
