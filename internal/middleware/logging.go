package middleware

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func LogRequest() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get("User-Agent")
			// TODO: check why logger is not logging
			log.Tracef(" ====> request [%s] path: [%s] [UA: %s]", r.Method, r.URL.Path, userAgent)
			fmt.Printf(" ====> request [%s] path: [%s] [UA: %s]\n", r.Method, r.URL.Path, userAgent)
			next.ServeHTTP(w, r)
		})
	}
}
