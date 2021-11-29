package middleware

import (
	"io"
	"net/http"
)

// DrainAndCloseRequest - avoid potential overhead and memory leaks by draining the request body and closing it
func DrainAndCloseRequest() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
			if r.Body != nil {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()
			}
		})
	}
}
