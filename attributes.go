package gintelemetry

import "go.opentelemetry.io/otel/attribute"

// AttributeAPI provides unified attribute builders that work across all telemetry types.
// Attributes created with this API can be used for tracing, metrics, and logging.
type AttributeAPI struct{}

// String creates a string-valued attribute.
func (AttributeAPI) String(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// Int creates an int-valued attribute.
func (AttributeAPI) Int(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

// Int64 creates an int64-valued attribute.
func (AttributeAPI) Int64(key string, value int64) attribute.KeyValue {
	return attribute.Int64(key, value)
}

// Float64 creates a float64-valued attribute.
func (AttributeAPI) Float64(key string, value float64) attribute.KeyValue {
	return attribute.Float64(key, value)
}

// Bool creates a bool-valued attribute.
func (AttributeAPI) Bool(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}

// Strings creates a string slice-valued attribute.
func (AttributeAPI) Strings(key string, values []string) attribute.KeyValue {
	return attribute.StringSlice(key, values)
}
