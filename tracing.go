package gintelemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("gintelemetry")

// TraceAPI provides functions for distributed tracing.
type TraceAPI struct{}

// Trace is the namespace for all tracing operations.
var Trace = TraceAPI{}

// Attribute represents a key-value pair for telemetry metadata.
type Attribute = attribute.KeyValue

// StartSpan starts a new span and returns a context with the span and a stop function.
// The caller MUST defer the stop function.
func (TraceAPI) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, func()) {
	ctx, span := tracer.Start(ctx, name, opts...)
	return ctx, func() { span.End() }
}

// StartSpanWithAttributes starts a new span with the given name and attributes.
func (TraceAPI) StartSpanWithAttributes(ctx context.Context, name string, attrs ...Attribute) (context.Context, func()) {
	return Trace.StartSpan(ctx, name, trace.WithAttributes(attrs...))
}

// RecordError records an error on the span associated with the context.
func (TraceAPI) RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetAttributes sets attributes on the span associated with the context.
func (TraceAPI) SetAttributes(ctx context.Context, attrs ...Attribute) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddEvent adds an event to the span associated with the context.
func (TraceAPI) AddEvent(ctx context.Context, name string, attrs ...Attribute) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SetStatus sets the status of the span associated with the context.
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

// Attribute helper functions for creating telemetry metadata.

// String creates a string attribute.
func (TraceAPI) String(key, value string) Attribute {
	return attribute.String(key, value)
}

// Int creates an int attribute.
func (TraceAPI) Int(key string, value int) Attribute {
	return attribute.Int(key, value)
}

// Int64 creates an int64 attribute.
func (TraceAPI) Int64(key string, value int64) Attribute {
	return attribute.Int64(key, value)
}

// Float64 creates a float64 attribute.
func (TraceAPI) Float64(key string, value float64) Attribute {
	return attribute.Float64(key, value)
}

// Bool creates a boolean attribute.
func (TraceAPI) Bool(key string, value bool) Attribute {
	return attribute.Bool(key, value)
}

// Strings creates a string slice attribute.
func (TraceAPI) Strings(key string, values []string) Attribute {
	return attribute.StringSlice(key, values)
}

// Status codes for spans.
const (
	StatusUnset = codes.Unset
	StatusError = codes.Error
	StatusOK    = codes.Ok
)
