package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
)

func PanicRecovery(instr *metrics.Manager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("http: panic serving %s: %v\n%s", req.URL.Path, r, debug.Stack())
					instr.CounterHandleRequestPanic.Inc()
				}
			}()

			// handler call
			next.ServeHTTP(respWriter, req)
		})
	}
}
