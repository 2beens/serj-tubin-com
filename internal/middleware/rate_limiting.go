package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/2beens/serjtubincom/internal/telemetry/metrics"

	"github.com/go-redis/redis_rate/v9"
)

type RequestRateLimiter interface {
	Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error)
}

func RateLimit(
	rateLimiter RequestRateLimiter,
	routerName string,
	allowedPerMin int,
	metricsManager *metrics.Manager,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			res, err := rateLimiter.Allow(
				r.Context(),
				routerName,
				redis_rate.PerMinute(allowedPerMin),
			)
			if err != nil {
				http.Error(w, "rate limit internal error", http.StatusInternalServerError)
				return
			}

			if res.Allowed > 0 {
				next.ServeHTTP(w, r)
				return
			}

			metricsManager.CounterRateLimitedRequests.Inc()
			http.Error(
				w,
				fmt.Sprintf("retry after %f seconds", res.RetryAfter.Seconds()),
				http.StatusTooEarly,
			)
		})
	}
}
