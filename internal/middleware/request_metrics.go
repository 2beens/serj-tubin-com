package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/instrumentation"
	"github.com/prometheus/client_golang/prometheus"
)

func RequestMetrics(instr *instrumentation.Instrumentation) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
			defer func(begin time.Time) {
				instr.HistRequestDuration.Observe(time.Since(begin).Seconds())
			}(time.Now())

			resp := &responseWriter{respWriter, http.StatusOK}

			// handler call
			next.ServeHTTP(resp, req)

			instr.CounterRequests.With(
				prometheus.Labels{
					"method": req.Method,
					"status": strconv.Itoa(resp.statusCode),
				},
			).Inc()
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.statusCode = statusCode
}
