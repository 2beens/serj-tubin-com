package tracing

import "go.opentelemetry.io/otel"

var GlobalTracer = otel.Tracer("main-backend")
