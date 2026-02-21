package middleware

import (
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// DrainAndCloseRequest - avoid potential overhead and memory leaks by draining the request body and closing it
func DrainAndCloseRequest() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			if r.Body != nil {
				if _, err := io.Copy(io.Discard, r.Body); err != nil {
					log.Debugf("drain request body: %s", err)
				}
				if err := r.Body.Close(); err != nil {
					log.Debugf("close request body: %s", err)
				}
			}
		})
	}
}
