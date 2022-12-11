package tracing

import (
	"fmt"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var GlobalTracer = otel.Tracer("main-backend")
var GlobalNetlogBackupTracer = otel.Tracer("gdrive-netlog-backup")

// HoneycombSetup uses honeycomb distro to setup OpenTelemetry SDK
func HoneycombSetup(
	honeycombTracingEnabled bool,
	component string,
	redisClient *redis.Client,
) (func(), error) {
	if !honeycombTracingEnabled {
		return func() { /*noop*/ }, nil
	}

	if redisClient != nil {
		// tracing support for redis client
		redisClient.AddHook(
			redisotel.NewTracingHook(
				redisotel.WithAttributes(attribute.String("component", component)),
				redisotel.WithTracerProvider(otel.GetTracerProvider()),
			),
		)
	}

	// enable multi-span attributes
	// https://opentelemetry.io/docs/concepts/signals/baggage/
	bsp := honeycomb.NewBaggageSpanProcessor()

	shutdownFunc, err := launcher.ConfigureOpenTelemetry(
		launcher.WithLogLevel("info"), // info log is default anyway
		launcher.WithSpanProcessor(bsp),
	)
	if err != nil {
		return nil, fmt.Errorf("honecomb, configure open telemetry: %w", err)
	}

	return shutdownFunc, err
}
