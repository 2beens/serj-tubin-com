package middleware

import (
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
	"github.com/2beens/serjtubincom/pkg"
)

type AuthMiddlewareHandler struct {
	gymstatsIOSAppSecret  string
	browserRequestsSecret string
	loginChecker          *auth.LoginChecker
	allowedPaths          map[string]bool
	allowedPathsPrefixes  []string
}

func NewAuthMiddlewareHandler(
	gymstatsIOSAppSecret string,
	browserRequestsSecret string,
	loginChecker *auth.LoginChecker,
) *AuthMiddlewareHandler {
	return &AuthMiddlewareHandler{
		gymstatsIOSAppSecret:  gymstatsIOSAppSecret,
		browserRequestsSecret: browserRequestsSecret,
		loginChecker:          loginChecker,
		allowedPaths: map[string]bool{
			// blog handler:
			"/blog/all":  true,
			"/blog/clap": true,

			// misc handler:
			"/":             true,
			"/quote/random": true,
			"/whereami":     true,
			"/myip":         true,
			"/version":      true,

			// weather handler:
			"/weather/current":  true,
			"/weather/tomorrow": true,
			"/weather/5days":    true,

			// login-logout:
			"/a/login":  true,
			"/a/logout": true,
		},
		allowedPathsPrefixes: []string{
			"/blog/page/",
		},
	}
}

func (h *AuthMiddlewareHandler) pathIsAlwaysAllowed(path string) bool {
	if h.allowedPaths[path] {
		return true
	}
	for _, prefix := range h.allowedPathsPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (h *AuthMiddlewareHandler) AuthCheck() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracing.GlobalTracer.Start(r.Context(), "middleware.auth")
			defer span.End()

			if r.Method == http.MethodOptions {
				w.Header().Add("Allow", "GET, POST, OPTIONS")
				w.WriteHeader(http.StatusOK)
				span.SetStatus(codes.Ok, "options-ok")
				return
			}

			if h.pathIsAlwaysAllowed(r.URL.Path) {
				span.SetStatus(codes.Ok, "ok")
				next.ServeHTTP(w, r)
				return
			}

			// a non-standard req. header is set, and thus - browser makes a preflight/OPTIONS request:
			//	https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#preflighted_requests
			// TODO: use Authorization header, not this custom one
			authToken := r.Header.Get("X-SERJ-TOKEN")

			// visitor board: only path /messages/delete/ is protected
			if strings.HasPrefix(r.URL.Path, "/board/messages/") {
				if !strings.HasPrefix(r.URL.Path, "/board/messages/delete/") {
					span.SetStatus(codes.Ok, "ok")
					next.ServeHTTP(w, r)
					return
				}
			}

			// requests coming from browser extension
			if strings.HasPrefix(r.URL.Path, "/netlog/new") {
				if h.browserRequestsSecret != authToken {
					reqIp, _ := pkg.ReadUserIP(r)
					log.Errorf("unauthorized /netlog/new request detected from %s, authToken: %s", reqIp, authToken)
					// fool the "attacker" by a fake positive response
					pkg.WriteTextResponseOK(w, "added")
					span.SetStatus(codes.Error, "decoy-sent")
					return
				}
				span.SetStatus(codes.Ok, "ok")
				next.ServeHTTP(w, r)
				return
			}

			userAgent := r.Header.Get("User-Agent")
			isGymstatsAllowedAgent := strings.HasPrefix(userAgent, "curl/") ||
				strings.HasPrefix(userAgent, "GymStats/1") ||
				strings.HasPrefix(userAgent, "test-agent")
			if isGymstatsAllowedAgent && strings.HasPrefix(r.URL.Path, "/gymstats") {
				// requests coming from GymStats iOS app or curl
				receivedAuthToken := r.Header.Get("Authorization")
				if h.gymstatsIOSAppSecret != receivedAuthToken {
					http.Error(w, "no can do", http.StatusUnauthorized)
					span.SetStatus(codes.Error, "missing-auth-token")
					return
				}
				span.SetStatus(codes.Ok, "ok")
				next.ServeHTTP(w, r)
				return
			}

			if authToken == "" {
				log.Tracef("[missing token] [auth middleware] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				span.SetStatus(codes.Error, "missing-auth-token")
				return
			}

			isLogged, err := h.loginChecker.IsLogged(ctx, authToken)
			if err != nil {
				log.Errorf("[failed login check] => %s: %s", r.URL.Path, err)
				http.Error(w, "no can do", http.StatusUnauthorized)
				span.SetStatus(codes.Error, "check-logged-err")
				span.RecordError(err)
				return
			}
			if !isLogged {
				log.Tracef("[invalid token] [auth middleware] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				span.SetStatus(codes.Error, "not-logged")
				return
			}

			span.SetStatus(codes.Ok, "ok")
			next.ServeHTTP(w, r)
		})
	}
}
