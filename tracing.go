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

// StartSpan creates a new span and returns a context containing the span and a function to end it.
// The returned context should be used for all operations within this span's scope.
// Always call the returned function (typically with defer) to properly end the span.
//
// Example:
//
//	ctx, stop := tel.Trace().StartSpan(ctx, "database.query")
//	defer stop()
//	// ... perform database query ...
func (t TraceAPI) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, func()) {
	ctx, span := t.tracer.Start(ctx, name, opts...)
	return ctx, func() { span.End() }
}

// StartSpanWithAttributes creates a new span with attributes and returns a context and end function.
// This is a convenience method that combines StartSpan with initial attributes.
//
// Example:
//
//	ctx, stop := tel.Trace().StartSpanWithAttributes(ctx, "db.query",
//	    tel.Attr().String("db.system", "postgres"),
//	    tel.Attr().String("db.operation", "SELECT"),
//	)
//	defer stop()
func (t TraceAPI) StartSpanWithAttributes(ctx context.Context, name string, attrs ...Attribute) (context.Context, func()) {
	return t.StartSpan(ctx, name, trace.WithAttributes(attrs...))
}

// RecordError records an error in the current span if one exists and is recording.
// If no span exists or the span is not recording, this is a safe no-op.
// This method never returns an error itself and never panics.
//
// The error is recorded with the span status set to Error.
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
// If no span exists or the span is not recording, this is a safe no-op.
// The context must contain an active span for attributes to be recorded.
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
// Events represent significant points in time within a span's duration.
// If no span exists or the span is not recording, this is a safe no-op.
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
// Use this to mark a span as successful (StatusOK), failed (StatusError), or unset (StatusUnset).
// If no span exists or the span is not recording, this is a safe no-op.
//
// Example:
//
//	tel.Trace().SetStatus(ctx, gintelemetry.StatusOK, "operation completed successfully")
func (TraceAPI) SetStatus(ctx context.Context, code codes.Code, description string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// SpanFromContext returns the current span from the context.
// This is useful for advanced use cases where you need direct access to the span.
func (TraceAPI) SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// Status codes for span status
const (
	StatusUnset = codes.Unset
	StatusError = codes.Error
	StatusOK    = codes.Ok
)
