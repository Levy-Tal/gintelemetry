package provider

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.9.0"
)

type ProviderConfig struct {
	ServiceName      string
	GlobalAttributes map[string]string
}

type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

func NewResource(cfg ProviderConfig) (*resource.Resource, error) {
	attrs := make([]attribute.KeyValue, 0, 1+len(cfg.GlobalAttributes))
	attrs = append(attrs, semconv.ServiceNameKey.String(cfg.ServiceName))
	for k, v := range cfg.GlobalAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}
	return resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, attrs...))
}

func NewTracerProvider(ctx context.Context, exporter sdktrace.SpanExporter, res *resource.Resource) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
}

func NewMeterProvider(ctx context.Context, exporter sdkmetric.Exporter, res *resource.Resource) *sdkmetric.MeterProvider {
	return sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)), sdkmetric.WithResource(res))
}

func NewLoggerProvider(ctx context.Context, exporter sdklog.Exporter, res *resource.Resource) *sdklog.LoggerProvider {
	return sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)), sdklog.WithResource(res))
}

func (p *Providers) SetGlobalProviders() {
	if p.TracerProvider != nil {
		otel.SetTracerProvider(p.TracerProvider)
	}
	if p.MeterProvider != nil {
		otel.SetMeterProvider(p.MeterProvider)
	}
	if p.LoggerProvider != nil {
		global.SetLoggerProvider(p.LoggerProvider)
	}
}

func (p *Providers) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var errs []error
	if p.TracerProvider != nil {
		if err := p.TracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
		}
	}
	if p.LoggerProvider != nil {
		if err := p.LoggerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("logger shutdown: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (p *Providers) ForceFlush(ctx context.Context) error {
	if p == nil {
		return nil
	}
	var errs []error
	if p.TracerProvider != nil {
		if err := p.TracerProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer flush: %w", err))
		}
	}
	if p.MeterProvider != nil {
		if err := p.MeterProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter flush: %w", err))
		}
	}
	return errors.Join(errs...)
}
