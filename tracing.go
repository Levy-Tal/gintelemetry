package gintelemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TraceAPI struct {
	tracer trace.Tracer
}

type Attribute = attribute.KeyValue

// StartSpan creates a new span with optional attributes and returns a context containing the span and a function to end it.
// The returned context should be used for all operations within this span's scope.
// Always call the returned function (typically with defer) to properly end the span.
//
// Example:
//
//	ctx, stop := tel.Trace().StartSpan(ctx, "database.query",
//	    tel.Attr().String("db.table", "users"),
//	    tel.Attr().String("db.operation", "SELECT"),
//	)
//	defer stop()
func (t TraceAPI) StartSpan(ctx context.Context, name string, attrs ...Attribute) (context.Context, func()) {
	opts := []trace.SpanStartOption{}
	if len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}
	ctx, span := t.tracer.Start(ctx, name, opts...)
	return ctx, func() { span.End() }
}

// StartSpanWithKind creates a new span with a specific kind and optional attributes.
// Use this for background jobs, client calls, or other non-server spans.
//
// Example:
//
//	ctx, stop := tel.Trace().StartSpanWithKind(ctx, "worker.job",
//	    gintelemetry.SpanKindInternal,
//	    tel.Attr().String("job.name", "scraper"),
//	)
//	defer stop()
func (t TraceAPI) StartSpanWithKind(ctx context.Context, name string, kind SpanKind, attrs ...Attribute) (context.Context, func()) {
	opts := []trace.SpanStartOption{trace.WithSpanKind(trace.SpanKind(kind))}
	if len(attrs) > 0 {
		opts = append(opts, trace.WithAttributes(attrs...))
	}
	ctx, span := t.tracer.Start(ctx, name, opts...)
	return ctx, func() { span.End() }
}

// RecordError records an error in the current span if one exists and is recording.
// If no span exists or the span is not recording, this is a safe no-op.
//
// Example:
//
//	if err != nil {
//	    tel.Trace().RecordError(ctx, err)
//	    return err
//	}
func (TraceAPI) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetAttributes adds attributes to the current span if one exists and is recording.
//
// Example:
//
//	tel.Trace().SetAttributes(ctx,
//	    tel.Attr().String("user.id", userID),
//	    tel.Attr().Int("items.count", len(items)),
//	)
func (TraceAPI) SetAttributes(ctx context.Context, attrs ...Attribute) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddEvent adds an event to the current span if one exists and is recording.
//
// Example:
//
//	tel.Trace().AddEvent(ctx, "cache.miss",
//	    tel.Attr().String("cache.key", key),
//	)
func (TraceAPI) AddEvent(ctx context.Context, name string, attrs ...Attribute) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetStatus sets the status of the current span if one exists and is recording.
//
// Example:
//
//	tel.Trace().SetStatus(ctx, gintelemetry.StatusOK, "operation completed")
func (TraceAPI) SetStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// SpanFromContext returns the current span from the context.
func (TraceAPI) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// Status codes for span status
const (
	StatusUnset = codes.Unset
	StatusError = codes.Error
	StatusOK    = codes.Ok
)
