package tracing

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var GlobalTracer = otel.Tracer("main-backend")
var GlobalNetlogBackupTracer = otel.Tracer("gdrive-netlog-backup")

type PgxOtelTracer struct {
	tracer         trace.Tracer
	tracingEnabled bool
}

func NewPgxOtelTracer(tracingEnabled bool, tracer trace.Tracer) *PgxOtelTracer {
	return &PgxOtelTracer{
		tracingEnabled: tracingEnabled,
		tracer:         tracer,
	}
}

func (t *PgxOtelTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	if !t.tracingEnabled {
		return ctx
	}
	ctx, span := t.tracer.Start(ctx, "db.connectStart")
	defer span.End()
	return ctx
}

func (t *PgxOtelTracer) TraceConnectEnd(ctx context.Context, data pgx.TraceConnectEndData) {
	if !t.tracingEnabled {
		return
	}

	ctx, span := t.tracer.Start(ctx, "db.connectEnd")
	defer span.End()

	if data.Err != nil {
		span.SetStatus(codes.Error, data.Err.Error())
		span.RecordError(data.Err)
	}
}

func (t *PgxOtelTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !t.tracingEnabled {
		return ctx
	}

	ctx, span := t.tracer.Start(ctx, "db.queryStart")
	defer span.End()

	span.SetAttributes(attribute.String("sql", data.SQL))
	//span.SetAttributes(attribute.String("sql_args", data.Args))

	return ctx
}

func (t *PgxOtelTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if !t.tracingEnabled {
		return
	}

	ctx, span := t.tracer.Start(ctx, "db.queryEnd")
	defer span.End()

	span.SetAttributes(attribute.String("commandTag", data.CommandTag.String()))
	if data.Err != nil {
		span.SetStatus(codes.Error, data.Err.Error())
		span.RecordError(data.Err)
	}
}
