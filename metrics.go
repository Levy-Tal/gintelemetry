package gintelemetry

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

const maxCachedMetrics = 10000

type metricCacheKey struct {
	name       string
	valueType  string
	metricType string
}

type metricCacheEntry struct {
	metric   any
	lastUsed atomic.Int64
}

// MetricAPI provides functions for recording metrics.
//
// IMPORTANT: Metrics are cached internally (up to 10,000 unique combinations).
// Do NOT use dynamic values in metric names. Use attributes instead:
//
//	✗ BAD:  tel.Metric().Counter("user_" + userID).Add(ctx, 1)
//	✓ GOOD: tel.Metric().IncrementCounter(ctx, "user_events", tel.Attr().String("user.id", userID))
//
// If you exceed the cache limit, metrics will still work but won't be cached,
// which may impact performance.
//
// Context Usage:
// All metric recording methods accept a context for trace correlation.
// If the context contains an active span, metrics can be correlated with traces.
// Using context.Background() is safe but loses correlation benefits.
type MetricAPI struct {
	meter        metric.Meter
	logger       *slog.Logger
	counterCache *sync.Map
	gaugeCache   *sync.Map
	histoCache   *sync.Map
	cacheSize    *atomic.Int64
}

type MetricAttribute = attribute.KeyValue

func (m MetricAPI) getOrCreateMetric(
	name, valueType, metricType string,
	cache *sync.Map,
	create func(string) (any, error),
) any {
	key := metricCacheKey{name: name, valueType: valueType, metricType: metricType}

	if cached, ok := cache.Load(key); ok {
		if entry, ok := cached.(*metricCacheEntry); ok {
			entry.lastUsed.Store(time.Now().UnixNano())
			return entry.metric
		}
		if m.logger != nil {
			m.logger.Error("metric cache type mismatch", "name", name, "type", metricType)
		}
		cache.Delete(key)
		m.cacheSize.Add(-1)
	}

	if m.cacheSize.Load() >= maxCachedMetrics {
		if m.logger != nil {
			m.logger.Error("metric cache limit exceeded", "name", name, "limit", maxCachedMetrics)
		}
		metric, _ := create(name)
		return metric
	}

	metric, err := create(name)
	if err != nil && m.logger != nil {
		m.logger.Warn("failed to create metric", "name", name, "type", metricType, "error", err)
	}

	entry := &metricCacheEntry{metric: metric}
	entry.lastUsed.Store(time.Now().UnixNano())

	actual, loaded := cache.LoadOrStore(key, entry)
	if !loaded {
		m.cacheSize.Add(1)
	}
	if actualEntry, ok := actual.(*metricCacheEntry); ok {
		return actualEntry.metric
	}
	return metric
}

func (m MetricAPI) Counter(name string, opts ...metric.Int64CounterOption) metric.Int64Counter {
	result := m.getOrCreateMetric(name, "int64", "counter", m.counterCache, func(n string) (any, error) {
		counter, err := m.meter.Int64Counter(n, opts...)
		if err != nil {
			noopMeter := noop.NewMeterProvider().Meter("")
			counter, _ = noopMeter.Int64Counter(n)
		}
		return counter, err
	})
	return result.(metric.Int64Counter)
}

func (m MetricAPI) Histogram(name string, opts ...metric.Int64HistogramOption) metric.Int64Histogram {
	result := m.getOrCreateMetric(name, "int64", "histogram", m.histoCache, func(n string) (any, error) {
		histogram, err := m.meter.Int64Histogram(n, opts...)
		if err != nil {
			noopMeter := noop.NewMeterProvider().Meter("")
			histogram, _ = noopMeter.Int64Histogram(n)
		}
		return histogram, err
	})
	return result.(metric.Int64Histogram)
}

func (m MetricAPI) Gauge(name string, opts ...metric.Int64GaugeOption) metric.Int64Gauge {
	result := m.getOrCreateMetric(name, "int64", "gauge", m.gaugeCache, func(n string) (any, error) {
		gauge, err := m.meter.Int64Gauge(n, opts...)
		if err != nil {
			noopMeter := noop.NewMeterProvider().Meter("")
			gauge, _ = noopMeter.Int64Gauge(n)
		}
		return gauge, err
	})
	return result.(metric.Int64Gauge)
}

func (m MetricAPI) Float64Counter(name string, opts ...metric.Float64CounterOption) metric.Float64Counter {
	result := m.getOrCreateMetric(name, "float64", "counter", m.counterCache, func(n string) (any, error) {
		counter, err := m.meter.Float64Counter(n, opts...)
		if err != nil {
			noopMeter := noop.NewMeterProvider().Meter("")
			counter, _ = noopMeter.Float64Counter(n)
		}
		return counter, err
	})
	return result.(metric.Float64Counter)
}

func (m MetricAPI) Float64Histogram(name string, opts ...metric.Float64HistogramOption) metric.Float64Histogram {
	result := m.getOrCreateMetric(name, "float64", "histogram", m.histoCache, func(n string) (any, error) {
		histogram, err := m.meter.Float64Histogram(n, opts...)
		if err != nil {
			noopMeter := noop.NewMeterProvider().Meter("")
			histogram, _ = noopMeter.Float64Histogram(n)
		}
		return histogram, err
	})
	return result.(metric.Float64Histogram)
}

func (m MetricAPI) Float64Gauge(name string, opts ...metric.Float64GaugeOption) metric.Float64Gauge {
	result := m.getOrCreateMetric(name, "float64", "gauge", m.gaugeCache, func(n string) (any, error) {
		gauge, err := m.meter.Float64Gauge(n, opts...)
		if err != nil {
			noopMeter := noop.NewMeterProvider().Meter("")
			gauge, _ = noopMeter.Float64Gauge(n)
		}
		return gauge, err
	})
	return result.(metric.Float64Gauge)
}

// IncrementCounter increments a counter by 1 with optional attributes.
// This is a convenience method equivalent to Counter(name).Add(ctx, 1, metric.WithAttributes(attrs...)).
func (m MetricAPI) IncrementCounter(ctx context.Context, name string, attrs ...MetricAttribute) {
	m.Counter(name).Add(ctx, 1, metric.WithAttributes(attrs...))
}

// AddCounter adds a value to a counter with optional attributes.
// This is a convenience method equivalent to Counter(name).Add(ctx, value, metric.WithAttributes(attrs...)).
func (m MetricAPI) AddCounter(ctx context.Context, name string, value int64, attrs ...MetricAttribute) {
	m.Counter(name).Add(ctx, value, metric.WithAttributes(attrs...))
}

// RecordHistogram records a histogram value with optional attributes.
// This is a convenience method equivalent to Histogram(name).Record(ctx, value, metric.WithAttributes(attrs...)).
func (m MetricAPI) RecordHistogram(ctx context.Context, name string, value int64, attrs ...MetricAttribute) {
	m.Histogram(name).Record(ctx, value, metric.WithAttributes(attrs...))
}

// RecordDuration records a duration as milliseconds to a histogram with optional attributes.
// This is a convenience method that converts the duration to milliseconds and records it.
func (m MetricAPI) RecordDuration(ctx context.Context, name string, duration time.Duration, attrs ...MetricAttribute) {
	m.Histogram(name).Record(ctx, duration.Milliseconds(), metric.WithAttributes(attrs...))
}

// RecordGauge records a gauge value with optional attributes.
// This is a convenience method equivalent to Gauge(name).Record(ctx, value, metric.WithAttributes(attrs...)).
func (m MetricAPI) RecordGauge(ctx context.Context, name string, value int64, attrs ...MetricAttribute) {
	m.Gauge(name).Record(ctx, value, metric.WithAttributes(attrs...))
}

// AddFloat64Counter adds a value to a float64 counter with optional attributes.
// This is a convenience method equivalent to Float64Counter(name).Add(ctx, value, metric.WithAttributes(attrs...)).
func (m MetricAPI) AddFloat64Counter(ctx context.Context, name string, value float64, attrs ...MetricAttribute) {
	m.Float64Counter(name).Add(ctx, value, metric.WithAttributes(attrs...))
}

// RecordFloat64Histogram records a float64 histogram value with optional attributes.
// This is a convenience method equivalent to Float64Histogram(name).Record(ctx, value, metric.WithAttributes(attrs...)).
func (m MetricAPI) RecordFloat64Histogram(ctx context.Context, name string, value float64, attrs ...MetricAttribute) {
	m.Float64Histogram(name).Record(ctx, value, metric.WithAttributes(attrs...))
}

// RecordFloat64Gauge records a float64 gauge value with optional attributes.
// This is a convenience method equivalent to Float64Gauge(name).Record(ctx, value, metric.WithAttributes(attrs...)).
func (m MetricAPI) RecordFloat64Gauge(ctx context.Context, name string, value float64, attrs ...MetricAttribute) {
	m.Float64Gauge(name).Record(ctx, value, metric.WithAttributes(attrs...))
}
