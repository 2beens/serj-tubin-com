package middleware

import (
	"net/http"
	"strings"
)

var allowedOrigins = map[string]bool{
	"https://www.serj-tubin.com": true,
	"http://localhost:8080":      true,
	"test":                       true,
}

func Cors() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			switch {
			case
				allowedOrigins[origin],
				strings.HasPrefix(origin, "chrome-extension://"),
				// allow CORS to the files-box /link endpoint from anywhere
				strings.HasPrefix(r.URL.Path, "/link/"):
				{
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Headers",
						"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-SERJ-TOKEN",
					)
					w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
				}
			default:
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
