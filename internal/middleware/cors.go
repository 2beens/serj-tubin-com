package middleware

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

var allowedOrigins = map[string]bool{
	"https://www.serj-tubin.com": true,
	"https://2beens.online":      true,
	"http://localhost:8080":      true,
	"test":                       true,
}

func Cors() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			userAgent := r.Header.Get("User-Agent")

			// used with spotify tracker
			if strings.HasPrefix(r.URL.Path, "/spotify/auth") && origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			switch {
			case
				allowedOrigins[origin],
				strings.HasPrefix(origin, "chrome-extension://"),
				strings.HasPrefix(userAgent, "GymStats/1"),
				strings.HasPrefix(userAgent, "curl/"),
				strings.HasPrefix(userAgent, "test-agent"),
				// allow CORS to the files-box /link endpoint from anywhere
				strings.HasPrefix(r.URL.Path, "/link/"),
				// allow CORS to the gymstats image endpoint from anywhere
				strings.HasPrefix(r.URL.Path, "/gymstats/image/"),
				// allow MCP endpoint (Cursor and other MCP clients often send no Origin)
				strings.HasPrefix(r.URL.Path, "/mcp"):
				{
					allowOrigin := origin
					if allowOrigin == "" && strings.HasPrefix(r.URL.Path, "/mcp") {
						allowOrigin = "*"
					}
					w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
					w.Header().Set("Access-Control-Allow-Headers",
						"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-SERJ-TOKEN, X-MCP-Secret, MCP-Protocol-Version, MCP-Session-Id",
					)
					w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
				}
			default:
				log.Warnf("CORS: origin not allowed for path [%s] and origin [%s]", r.URL.Path, origin)
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
