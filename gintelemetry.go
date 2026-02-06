// Package gintelemetry provides simple OpenTelemetry bootstrap for Gin applications.
package gintelemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Telemetry struct {
	serviceName     string
	tracerProvider  *sdktrace.TracerProvider
	meterProvider   *sdkmetric.MeterProvider
	loggerProvider  *sdklog.LoggerProvider
	logger          *slog.Logger
	meter           metric.Meter
	tracer          trace.Tracer
	shutdownTimeout time.Duration
	shutdownOnce    sync.Once
	shutdownErr     error
	shutdownDone    chan struct{}
}

func Start(ctx context.Context, cfg Config) (*Telemetry, *gin.Engine, error) {
	if err := cfg.validate(); err != nil {
		return nil, nil, err
	}

	// Create resource with service name and global attributes
	attrs := []attribute.KeyValue{
		attribute.String("service.name", cfg.ServiceName),
	}
	for k, v := range cfg.GlobalAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("", attrs...),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporters based on protocol
	var traceExporter sdktrace.SpanExporter
	var metricExporter sdkmetric.Exporter
	var logExporter sdklog.Exporter

	if cfg.Protocol == ProtocolHTTP {
		// HTTP exporters
		traceOpts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		}
		traceExporter, err = otlptracehttp.New(ctx, traceOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
		}

		metricOpts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
		}
		metricExporter, err = otlpmetrichttp.New(ctx, metricOpts...)
		if err != nil {
			_ = traceExporter.Shutdown(ctx)
			return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
		}

		logOpts := []otlploghttp.Option{
			otlploghttp.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			logOpts = append(logOpts, otlploghttp.WithInsecure())
		}
		logExporter, err = otlploghttp.New(ctx, logOpts...)
		if err != nil {
			_ = traceExporter.Shutdown(ctx)
			_ = metricExporter.Shutdown(ctx)
			return nil, nil, fmt.Errorf("failed to create log exporter: %w", err)
		}
	} else {
		// gRPC exporters (default)
		traceOpts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
		}
		traceExporter, err = otlptracegrpc.New(ctx, traceOpts...)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
		}

		metricOpts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
		}
		metricExporter, err = otlpmetricgrpc.New(ctx, metricOpts...)
		if err != nil {
			_ = traceExporter.Shutdown(ctx)
			return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
		}

		logOpts := []otlploggrpc.Option{
			otlploggrpc.WithEndpoint(cfg.Endpoint),
		}
		if cfg.Insecure {
			logOpts = append(logOpts, otlploggrpc.WithInsecure())
		}
		logExporter, err = otlploggrpc.New(ctx, logOpts...)
		if err != nil {
			_ = traceExporter.Shutdown(ctx)
			_ = metricExporter.Shutdown(ctx)
			return nil, nil, fmt.Errorf("failed to create log exporter: %w", err)
		}
	}

	// Create providers
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)

	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	// Create logger with dual output (OTLP + stdout)
	logger := otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(loggerProvider))
	logger = applyLevelFilter(logger, cfg.getLogLevel())

	t := &Telemetry{
		serviceName:     cfg.ServiceName,
		tracerProvider:  tracerProvider,
		meterProvider:   meterProvider,
		loggerProvider:  loggerProvider,
		logger:          logger,
		meter:           meterProvider.Meter(cfg.ServiceName),
		tracer:          tracerProvider.Tracer(cfg.ServiceName),
		shutdownTimeout: cfg.getShutdownTimeout(),
		shutdownDone:    make(chan struct{}),
	}

	// Set global providers if requested
	if cfg.SetGlobalProvider {
		otel.SetTracerProvider(tracerProvider)
		otel.SetMeterProvider(meterProvider)
		global.SetLoggerProvider(loggerProvider)
	}

	// Create Gin router with recovery and tracing middleware
	router := gin.New()
	router.Use(gin.Recovery(), otelgin.Middleware(cfg.ServiceName,
		otelgin.WithTracerProvider(tracerProvider)))

	return t, router, nil
}

func (t *Telemetry) Flush(ctx context.Context) error {
	if t == nil {
		return nil
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.shutdownTimeout)
		defer cancel()
	}

	var errs []error
	if t.tracerProvider != nil {
		if err := t.tracerProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer flush: %w", err))
		}
	}
	if t.meterProvider != nil {
		if err := t.meterProvider.ForceFlush(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter flush: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil {
		return nil
	}

	t.shutdownOnce.Do(func() {
		defer close(t.shutdownDone)

		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, t.shutdownTimeout)
			defer cancel()
		}

		var errs []error
		if t.tracerProvider != nil {
			if err := t.tracerProvider.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
			}
		}
		if t.meterProvider != nil {
			if err := t.meterProvider.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("meter shutdown: %w", err))
			}
		}
		if t.loggerProvider != nil {
			if err := t.loggerProvider.Shutdown(ctx); err != nil {
				errs = append(errs, fmt.Errorf("logger shutdown: %w", err))
			}
		}

		t.shutdownErr = errors.Join(errs...)
	})

	return t.shutdownErr
}

func (t *Telemetry) ShutdownDone() <-chan struct{} {
	if t == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return t.shutdownDone
}

func (t *Telemetry) Log() LogAPI {
	return LogAPI{logger: t.logger}
}

func (t *Telemetry) Trace() TraceAPI {
	return TraceAPI{tracer: t.tracer}
}

func (t *Telemetry) Metric() MetricAPI {
	return MetricAPI{meter: t.meter}
}

// Attr returns the unified attribute builder for use across all telemetry types.
func (t *Telemetry) Attr() AttributeAPI {
	return AttributeAPI{}
}

func (t *Telemetry) TracerProvider() *sdktrace.TracerProvider {
	if t == nil {
		return nil
	}
	return t.tracerProvider
}

func (t *Telemetry) MeterProvider() *sdkmetric.MeterProvider {
	if t == nil {
		return nil
	}
	return t.meterProvider
}

func (t *Telemetry) LoggerProvider() *sdklog.LoggerProvider {
	if t == nil {
		return nil
	}
	return t.loggerProvider
}
