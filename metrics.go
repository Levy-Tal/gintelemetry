package gintelemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricAPI provides functions for recording metrics.
// This is a thin wrapper around OpenTelemetry's metric API.
type MetricAPI struct {
	meter metric.Meter
}

type MetricAttribute = attribute.KeyValue

// Counter returns an Int64Counter instrument.
// The OpenTelemetry SDK caches instruments internally, so calling this multiple
// times with the same name is safe and efficient.
func (m MetricAPI) Counter(name string, opts ...metric.Int64CounterOption) metric.Int64Counter {
	counter, _ := m.meter.Int64Counter(name, opts...)
	return counter
}

// Histogram returns an Int64Histogram instrument.
func (m MetricAPI) Histogram(name string, opts ...metric.Int64HistogramOption) metric.Int64Histogram {
	histogram, _ := m.meter.Int64Histogram(name, opts...)
	return histogram
}

// Gauge returns an Int64Gauge instrument.
func (m MetricAPI) Gauge(name string, opts ...metric.Int64GaugeOption) metric.Int64Gauge {
	gauge, _ := m.meter.Int64Gauge(name, opts...)
	return gauge
}

// Float64Counter returns a Float64Counter instrument.
func (m MetricAPI) Float64Counter(name string, opts ...metric.Float64CounterOption) metric.Float64Counter {
	counter, _ := m.meter.Float64Counter(name, opts...)
	return counter
}

// Float64Histogram returns a Float64Histogram instrument.
func (m MetricAPI) Float64Histogram(name string, opts ...metric.Float64HistogramOption) metric.Float64Histogram {
	histogram, _ := m.meter.Float64Histogram(name, opts...)
	return histogram
}

// Float64Gauge returns a Float64Gauge instrument.
func (m MetricAPI) Float64Gauge(name string, opts ...metric.Float64GaugeOption) metric.Float64Gauge {
	gauge, _ := m.meter.Float64Gauge(name, opts...)
	return gauge
}

// AddCounter adds to counter with attributes.
func (m MetricAPI) AddCounter(ctx context.Context, name string, value int64, attrs ...Attribute) {
	m.Counter(name).Add(ctx, value, metric.WithAttributes(attrs...))
}

// RecordHistogram records histogram with attributes.
func (m MetricAPI) RecordHistogram(ctx context.Context, name string, value int64, attrs ...Attribute) {
	m.Histogram(name).Record(ctx, value, metric.WithAttributes(attrs...))
}

// RecordGauge records gauge with attributes.
func (m MetricAPI) RecordGauge(ctx context.Context, name string, value int64, attrs ...Attribute) {
	m.Gauge(name).Record(ctx, value, metric.WithAttributes(attrs...))
}

// AddFloat64Counter adds to float64 counter with attributes.
func (m MetricAPI) AddFloat64Counter(ctx context.Context, name string, value float64, attrs ...Attribute) {
	m.Float64Counter(name).Add(ctx, value, metric.WithAttributes(attrs...))
}

// RecordFloat64Histogram records float64 histogram with attributes.
func (m MetricAPI) RecordFloat64Histogram(ctx context.Context, name string, value float64, attrs ...Attribute) {
	m.Float64Histogram(name).Record(ctx, value, metric.WithAttributes(attrs...))
}

// RecordFloat64Gauge records float64 gauge with attributes.
func (m MetricAPI) RecordFloat64Gauge(ctx context.Context, name string, value float64, attrs ...Attribute) {
	m.Float64Gauge(name).Record(ctx, value, metric.WithAttributes(attrs...))
}
