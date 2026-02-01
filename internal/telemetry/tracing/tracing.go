package tracing

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/contrib/processors/baggagecopy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
)

var GlobalTracer = otel.Tracer("main-backend")

var GlobalNetlogBackupTracer = otel.Tracer("gdrive-netlog-backup")

// HoneycombSetup configures the OpenTelemetry SDK to send traces to Honeycomb.
func HoneycombSetup(
	honeycombTracingEnabled bool,
	honeycombConfig HoneycombConfig,
	component string,
	redisClient *redis.Client,
) (func(), error) {
	if !honeycombTracingEnabled {
		return func() { /*noop*/ }, nil
	}

	otlpExporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(normalizeOtlpEndpoint(honeycombConfig.OtlpEndpoint)),
		otlptracegrpc.WithHeaders(resolveOtlpHeaders(honeycombConfig)),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("otel, create otlp exporter: %w", err)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(semconv.ServiceName(component)),
	)
	if err != nil {
		return nil, fmt.Errorf("otel, create resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(baggagecopy.NewSpanProcessor(nil)),
		sdktrace.WithBatcher(otlpExporter),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	if redisClient != nil {
		// tracing support for redis client
		redisClient.AddHook(
			redisotel.NewTracingHook(
				redisotel.WithAttributes(attribute.String("component", component)),
				redisotel.WithTracerProvider(tracerProvider),
			),
		)
	}

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tracerProvider.Shutdown(ctx)
	}, nil
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

type HoneycombConfig struct {
	OtlpEndpoint string
	OtlpHeaders  string
	ApiKey       string
	Dataset      string
}

const defaultHoneycombOtlpEndpoint = "api.honeycomb.io:443"

const defaultHoneycombDataset = "live"

func ReadHoneycombConfig() HoneycombConfig {
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if strings.TrimSpace(otlpEndpoint) == "" {
		otlpEndpoint = defaultHoneycombOtlpEndpoint
	}

	dataset := os.Getenv("HONEYCOMB_DATASET")
	if strings.TrimSpace(dataset) == "" {
		dataset = defaultHoneycombDataset
	}

	return HoneycombConfig{
		OtlpEndpoint: otlpEndpoint,
		OtlpHeaders:  os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"),
		ApiKey:       os.Getenv("HONEYCOMB_API_KEY"),
		Dataset:      dataset,
	}
}

func resolveOtlpHeaders(config HoneycombConfig) map[string]string {
	if config.OtlpHeaders != "" {
		return parseOtlpHeaders(config.OtlpHeaders)
	}

	headers := map[string]string{}
	if config.ApiKey != "" {
		headers["x-honeycomb-team"] = config.ApiKey
	}
	if config.Dataset != "" {
		headers["x-honeycomb-dataset"] = config.Dataset
	}

	return headers
}

func parseOtlpHeaders(rawHeaders string) map[string]string {
	headers := map[string]string{}
	for _, chunk := range strings.Split(rawHeaders, ",") {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		parts := strings.SplitN(chunk, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			continue
		}
		headers[key] = value
	}

	return headers
}

func normalizeOtlpEndpoint(rawEndpoint string) string {
	if rawEndpoint == "" {
		return ""
	}

	normalized := rawEndpoint
	if strings.HasPrefix(rawEndpoint, "http://") || strings.HasPrefix(rawEndpoint, "https://") {
		parsed, err := url.Parse(rawEndpoint)
		if err == nil && parsed.Host != "" {
			normalized = parsed.Host
		}
	}

	if !strings.Contains(normalized, ":") {
		normalized += ":443"
	}

	return normalized
}

func ValidateHoneycombConfig(config HoneycombConfig) error {
	if strings.TrimSpace(config.OtlpHeaders) == "" && strings.TrimSpace(config.ApiKey) == "" {
		return fmt.Errorf("otel, missing OTEL_EXPORTER_OTLP_HEADERS or HONEYCOMB_API_KEY")
	}

	if config.OtlpHeaders != "" {
		headers := parseOtlpHeaders(config.OtlpHeaders)
		if _, ok := headers["x-honeycomb-team"]; !ok {
			return fmt.Errorf("otel, OTEL_EXPORTER_OTLP_HEADERS missing x-honeycomb-team")
		}
	}

	return nil
}
