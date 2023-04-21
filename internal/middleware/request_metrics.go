package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func RequestMetrics(metricsManager *metrics.Manager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(respWriter http.ResponseWriter, req *http.Request) {
			statusCode := http.StatusOK
			defer func(begin time.Time) {
				metricsManager.HistogramRequestDuration.WithLabelValues(
					req.URL.Path,
					req.Method,
					strconv.Itoa(statusCode),
				).Observe(time.Since(begin).Seconds())
			}(time.Now())

			resp := &responseWriter{respWriter, statusCode}

			// handler call
			next.ServeHTTP(resp, req)
			statusCode = resp.statusCode

			metricsManager.CounterRequests.With(
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
