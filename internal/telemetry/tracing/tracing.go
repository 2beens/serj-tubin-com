package tracing

import (
	"fmt"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/otel-config-go/otelconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var GlobalTracer = otel.Tracer("main-backend")
var GlobalNetlogBackupTracer = otel.Tracer("gdrive-netlog-backup")

// HoneycombSetup uses honeycomb distro to set up the OpenTelemetry SDK
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
	bsp := honeycomb.NewBaggageSpanProcessor()

	// use honeycomb distro to set up OpenTelemetry SDK
	shutdownFunc, err := otelconfig.ConfigureOpenTelemetry(
		otelconfig.WithServiceName(component),
		otelconfig.WithSpanProcessor(bsp),
		otelconfig.WithLogLevel("info"), // info log is default anyway
	)
	if err != nil {
		return nil, fmt.Errorf("honeycomb, configure open telemetry: %w", err)
	}

	return shutdownFunc, err
}

// EndSpanWithErrCheck ends the span and sets the status to error if err is not nil
// otherwise sets the status to ok
// can be used as a defer function:
//
//	defer func() {
//	    EndSpanWithErrCheck(span, err)
//	}()
func EndSpanWithErrCheck(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "ok")
	}
	span.End()
}
