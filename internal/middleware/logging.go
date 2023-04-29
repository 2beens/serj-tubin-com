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
			log.WithFields(log.Fields{
				"method":       r.Method,
				"path":         r.URL.Path,
				"content_type": contentType,
				"user_agent":   userAgent,
			}).Tracef(
				" ====> request [%s %s]",
				r.Method, r.URL.Path,
			)
			next.ServeHTTP(w, r)
		})
	}
}
