package tracing

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/contrib/propagators/ot"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"

	// NOTE: this import is super important as applies the Honeycomb configuration to the launcher
	_ "github.com/honeycombio/honeycomb-opentelemetry-go"
	"github.com/honeycombio/opentelemetry-go-contrib/launcher"
)

var GlobalTracer = otel.Tracer("main-backend")
var GlobalNetlogBackupTracer = otel.Tracer("gdrive-netlog-backup")

// HoneycombSetup uses honeycomb distro to setup OpenTelemetry SDK
func HoneycombSetup(honeycombTracingEnabled bool, component string, redisClient *redis.Client) (func(), error) {
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

	shutdownFunc, err := launcher.ConfigureOpenTelemetry(
		launcher.WithLogLevel("info"), // info log is default anyway
	)
	if err != nil {
		return nil, fmt.Errorf("honecomb, configure open telemetry: %w", err)
	}

	return shutdownFunc, err
}

// GetDefaultTraceResource TODO: this func still not used anywhere
// will be used if I decide to setup otel tracing manually, instead of via honeycomb
func GetDefaultTraceResource(
	ctx context.Context,
	serviceName, serviceVersion string,
	resourceAttributes map[string]string,
) (*resource.Resource, error) {
	r := resource.Environment()

	hostnameSet := false
	for iter := r.Iter(); iter.Next(); {
		if iter.Attribute().Key == semconv.HostNameKey && len(iter.Attribute().Value.Emit()) > 0 {
			hostnameSet = true
		}
	}

	attributes := []attribute.KeyValue{
		semconv.TelemetrySDKNameKey.String("launcher"),
		semconv.TelemetrySDKLanguageGo,
	}

	if serviceName != "" {
		attributes = append(attributes, semconv.ServiceNameKey.String(serviceName))
	}

	if serviceVersion != "" {
		attributes = append(attributes, semconv.ServiceVersionKey.String(serviceVersion))
	}

	for key, value := range resourceAttributes {
		if len(value) > 0 {
			if key == string(semconv.HostNameKey) {
				hostnameSet = true
			}
			attributes = append(attributes, attribute.String(key, value))
		}
	}

	if !hostnameSet {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("unable to set host.name. Set OTEL_RESOURCE_ATTRIBUTES=\"host.name=<your_host_name>\" env var or configure WithResourceAttributes in code: %w", err)
		}
		attributes = append(attributes, semconv.HostNameKey.String(hostname))
	}

	attributes = append(r.Attributes(), attributes...)

	var err error
	r, err = resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithAttributes(attributes...),
	)
	if err != nil {
		return nil, err
	}

	// Note: There are new detectors we may wish to take advantage
	// of, now available in the default SDK (e.g., WithProcess(),
	// WithOSType(), ...).
	return r, nil
}

// SetupTracing can be used as:
// defResource, err := tracing.GetDefaultTraceResource(
//
//		ctx, "serj-tubin-com", "v1", map[string]string{},
//	)
//	otelShutdown, err := tracing.SetupTracing(ctx, defResource)
func SetupTracing(
	ctx context.Context,
	resources ...*resource.Resource,
) (func(timeoutCtx context.Context) error, error) {
	// FIXME: just does not work... does not send traces, not sure why

	honeycombApiKey := os.Getenv("HONEYCOMB_API_KEY")
	if honeycombApiKey == "" {
		return nil, errors.New("honeycomb api key not set")
	}

	opts := []trace.TracerProviderOption{
		//trace.WithSampler(c.Sampler),
		trace.WithSampler(trace.AlwaysSample()),
	}

	for _, r := range resources {
		opts = append(opts, trace.WithResource(r))
	}

	spanProcessors := []trace.SpanProcessor{}
	for _, sp := range spanProcessors {
		opts = append(opts, trace.WithSpanProcessor(sp))
	}

	// make sure the exporter is added last
	endpoint, insecure := "api.honeycomb.io:443", false
	spanExporter, err := newGRPCTraceExporter(ctx, endpoint, insecure, map[string]string{
		"x-honeycomb-team": honeycombApiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create span exporter: %v", err)
	}

	bsp := trace.NewBatchSpanProcessor(spanExporter)
	opts = append(opts, trace.WithSpanProcessor(bsp))

	tp := trace.NewTracerProvider(opts...)
	if err = configurePropagators([]string{
		"tracecontext", "baggage",
	}); err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tp)

	return func(timeoutCtx context.Context) error {
		_ = bsp.Shutdown(context.Background())
		return spanExporter.Shutdown(context.Background())
	}, nil
}

func newGRPCTraceExporter(ctx context.Context, endpoint string, insecure bool, headers map[string]string) (*otlptrace.Exporter, error) {
	secureOption := otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	if insecure {
		secureOption = otlptracegrpc.WithInsecure()
	}
	return otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithHeaders(headers),
			otlptracegrpc.WithCompressor(gzip.Name),
		),
	)
}

func configurePropagators(propagators []string) error {
	propagatorsMap := map[string]propagation.TextMapPropagator{
		"b3":           b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
		"baggage":      propagation.Baggage{},
		"tracecontext": propagation.TraceContext{},
		"ottrace":      ot.OT{},
	}
	var props []propagation.TextMapPropagator
	for _, key := range propagators {
		prop := propagatorsMap[key]
		if prop != nil {
			props = append(props, prop)
		}
	}
	if len(props) == 0 {
		return fmt.Errorf("invalid configuration: unsupported propagators. Supported options: b3,baggage,tracecontext,ottrace")
	}
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		props...,
	))
	return nil
}
