package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"

	log "github.com/sirupsen/logrus"
)

func PanicRecovery(metricsManager *metrics.Manager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("http: panic serving %s: %v\n%s", req.URL.Path, r, debug.Stack())
					if metricsManager != nil {
						metricsManager.CounterHandleRequestPanic.Inc()
					}
				}
			}()

			// handler call
			next.ServeHTTP(respWriter, req)
		})
	}
}
