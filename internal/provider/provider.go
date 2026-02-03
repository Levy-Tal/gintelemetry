package provider

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ProviderConfig holds configuration for telemetry providers.
type ProviderConfig struct {
	// ServiceName is the name of the service.
	ServiceName string
	// GlobalAttributes are resource-level attributes added to all telemetry.
	GlobalAttributes map[string]string
}

// Providers holds all OpenTelemetry providers.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// NewResource creates a new OpenTelemetry resource with service information.
func NewResource(cfg ProviderConfig) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
	}

	// Add global attributes
	for k, v := range cfg.GlobalAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			attrs...,
		),
	)
}

// NewTracerProvider creates a new tracer provider.
func NewTracerProvider(ctx context.Context, exporter sdktrace.SpanExporter, res *resource.Resource) *sdktrace.TracerProvider {
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp
}

// NewMeterProvider creates a new meter provider.
func NewMeterProvider(ctx context.Context, exporter sdkmetric.Exporter, res *resource.Resource) *sdkmetric.MeterProvider {
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	return mp
}

// NewLoggerProvider creates a new logger provider.
func NewLoggerProvider(ctx context.Context, exporter sdklog.Exporter, res *resource.Resource) *sdklog.LoggerProvider {
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)
	return lp
}

// Shutdown gracefully shuts down all providers.
func (p *Providers) Shutdown(ctx context.Context) error {
	var errs []error

	if err := p.TracerProvider.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := p.MeterProvider.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := p.LoggerProvider.Shutdown(ctx); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
