package middleware

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func LogRequest() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent := r.Header.Get("User-Agent")
			contentType := r.Header.Get("Content-Type")
			log.Tracef(
				" ====> request [%s] [path: %s] [content-type: %s] [UA: %s]",
				r.Method, r.URL.Path, contentType, userAgent,
			)
			next.ServeHTTP(w, r)
		})
	}
}
