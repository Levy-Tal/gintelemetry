package gintelemetry

import "go.opentelemetry.io/otel/trace"

// SpanKind represents the role of a span
type SpanKind = trace.SpanKind

const (
	SpanKindInternal = trace.SpanKindInternal
	SpanKindServer   = trace.SpanKindServer
	SpanKindClient   = trace.SpanKindClient
	SpanKindProducer = trace.SpanKindProducer
	SpanKindConsumer = trace.SpanKindConsumer
)
