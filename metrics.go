package gintelemetry

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter        = otel.Meter("gintelemetry")
	counterCache sync.Map
	gaugeCache   sync.Map
	histoCache   sync.Map
)

// MetricAPI provides functions for recording metrics.
type MetricAPI struct{}

// Metric is the namespace for all metrics operations.
var Metric = MetricAPI{}

// Attribute represents a key-value pair for telemetry metadata.
type MetricAttribute = attribute.KeyValue

// Counter returns or creates an Int64 counter. Counters are cached.
func (MetricAPI) Counter(name string, opts ...metric.Int64CounterOption) metric.Int64Counter {
	if c, ok := counterCache.Load(name); ok {
		return c.(metric.Int64Counter)
	}
	counter, _ := meter.Int64Counter(name, opts...)
	counterCache.Store(name, counter)
	return counter
}

// Histogram returns or creates an Int64 histogram. Histograms are cached.
func (MetricAPI) Histogram(name string, opts ...metric.Int64HistogramOption) metric.Int64Histogram {
	if h, ok := histoCache.Load(name); ok {
		return h.(metric.Int64Histogram)
	}
	histogram, _ := meter.Int64Histogram(name, opts...)
	histoCache.Store(name, histogram)
	return histogram
}

// Gauge returns or creates an Int64 gauge. Gauges are cached.
func (MetricAPI) Gauge(name string, opts ...metric.Int64GaugeOption) metric.Int64Gauge {
	if g, ok := gaugeCache.Load(name); ok {
		return g.(metric.Int64Gauge)
	}
	gauge, _ := meter.Int64Gauge(name, opts...)
	gaugeCache.Store(name, gauge)
	return gauge
}

// Float64Counter returns or creates a Float64 counter. Counters are cached.
func (MetricAPI) Float64Counter(name string, opts ...metric.Float64CounterOption) metric.Float64Counter {
	key := "float64:" + name
	if c, ok := counterCache.Load(key); ok {
		return c.(metric.Float64Counter)
	}
	counter, _ := meter.Float64Counter(name, opts...)
	counterCache.Store(key, counter)
	return counter
}

// Float64Histogram returns or creates a Float64 histogram. Histograms are cached.
func (MetricAPI) Float64Histogram(name string, opts ...metric.Float64HistogramOption) metric.Float64Histogram {
	key := "float64:" + name
	if h, ok := histoCache.Load(key); ok {
		return h.(metric.Float64Histogram)
	}
	histogram, _ := meter.Float64Histogram(name, opts...)
	histoCache.Store(key, histogram)
	return histogram
}

// Float64Gauge returns or creates a Float64 gauge. Gauges are cached.
func (MetricAPI) Float64Gauge(name string, opts ...metric.Float64GaugeOption) metric.Float64Gauge {
	key := "float64:" + name
	if g, ok := gaugeCache.Load(key); ok {
		return g.(metric.Float64Gauge)
	}
	gauge, _ := meter.Float64Gauge(name, opts...)
	gaugeCache.Store(key, gauge)
	return gauge
}

// Attribute helper functions for metrics.

// String creates a string attribute.
func (MetricAPI) String(key, value string) MetricAttribute {
	return attribute.String(key, value)
}

// Int creates an int attribute.
func (MetricAPI) Int(key string, value int) MetricAttribute {
	return attribute.Int(key, value)
}

// Int64 creates an int64 attribute.
func (MetricAPI) Int64(key string, value int64) MetricAttribute {
	return attribute.Int64(key, value)
}

// Float64 creates a float64 attribute.
func (MetricAPI) Float64(key string, value float64) MetricAttribute {
	return attribute.Float64(key, value)
}

// Bool creates a boolean attribute.
func (MetricAPI) Bool(key string, value bool) MetricAttribute {
	return attribute.Bool(key, value)
}
