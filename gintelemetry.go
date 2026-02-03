// Package gintelemetry provides opinionated OpenTelemetry bootstrap for Gin applications.
package gintelemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Levy-Tal/gintelemetry/internal/exporter"
	"github.com/Levy-Tal/gintelemetry/internal/provider"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	metricEvictionInterval = 10 * time.Minute
	metricTTL              = 1 * time.Hour
)

type Telemetry struct {
	serviceName     string
	providers       *provider.Providers
	logger          *slog.Logger
	meter           metric.Meter
	tracer          trace.Tracer
	shutdownTimeout time.Duration
	shutdownOnce    sync.Once
	shutdownErr     error
	shutdownDone    chan struct{}
	counterCache    sync.Map
	gaugeCache      sync.Map
	histoCache      sync.Map
	metricCacheSize atomic.Int64
	evictionCtx     context.Context
	evictionCancel  context.CancelFunc
	evictionDone    sync.WaitGroup
}

func Start(ctx context.Context, cfg Config) (*Telemetry, *gin.Engine, error) {
	cfg = cfg.copy()

	validatedCfg, err := cfg.validate()
	if err != nil {
		return nil, nil, err
	}

	exporterCfg := validatedCfg.buildExporterConfig()
	shutdownTimeout := validatedCfg.getShutdownTimeout()
	retries := validatedCfg.getExporterRetries()

	res, err := provider.NewResource(provider.ProviderConfig{
		ServiceName:      validatedCfg.ServiceName,
		GlobalAttributes: validatedCfg.GlobalAttributes,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	providers, cleanup, err := initializeProvidersWithCleanup(ctx, exporterCfg, res, shutdownTimeout, retries)
	defer func() {
		if providers == nil && cleanup != nil {
			_ = cleanup()
		}
	}()
	if err != nil {
		if cleanupErr := cleanup(); cleanupErr != nil {
			return nil, nil, fmt.Errorf("%w (cleanup error: %v)", err, cleanupErr)
		}
		return nil, nil, err
	}

	logger := otelslog.NewLogger(validatedCfg.ServiceName, otelslog.WithLoggerProvider(providers.LoggerProvider))
	logger = applyLevelFilter(logger, validatedCfg.getLogLevel())
	evictionCtx, evictionCancel := context.WithCancel(context.Background())

	t := &Telemetry{
		serviceName:     validatedCfg.ServiceName,
		providers:       providers,
		logger:          logger,
		meter:           providers.MeterProvider.Meter(validatedCfg.ServiceName),
		tracer:          providers.TracerProvider.Tracer(validatedCfg.ServiceName),
		shutdownTimeout: validatedCfg.getShutdownTimeout(),
		shutdownDone:    make(chan struct{}),
		evictionCtx:     evictionCtx,
		evictionCancel:  evictionCancel,
	}
	t.startMetricEviction()

	if validatedCfg.SetGlobalProvider {
		providers.SetGlobalProviders()
	}

	router := gin.New()
	router.Use(gin.Recovery(), otelgin.Middleware(validatedCfg.ServiceName,
		otelgin.WithTracerProvider(providers.TracerProvider)))

	return t, router, nil
}

func initializeProvidersWithCleanup(
	ctx context.Context,
	exporterCfg exporter.ExporterConfig,
	res *resource.Resource,
	shutdownTimeout time.Duration,
	retries int,
) (*provider.Providers, func() error, error) {
	var traceExp sdktrace.SpanExporter
	var metricExp sdkmetric.Exporter
	var logExp sdklog.Exporter
	var err error

	cleanup := func() error {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		var errs []error
		if traceExp != nil {
			if err := traceExp.Shutdown(cleanupCtx); err != nil {
				errs = append(errs, fmt.Errorf("trace exporter shutdown: %w", err))
			}
		}
		if metricExp != nil {
			if err := metricExp.Shutdown(cleanupCtx); err != nil {
				errs = append(errs, fmt.Errorf("metric exporter shutdown: %w", err))
			}
		}
		if logExp != nil {
			if err := logExp.Shutdown(cleanupCtx); err != nil {
				errs = append(errs, fmt.Errorf("log exporter shutdown: %w", err))
			}
		}
		return errors.Join(errs...)
	}

	if retries > 0 {
		traceExp, err = exporter.NewTraceExporterWithRetry(ctx, exporterCfg, retries)
	} else {
		traceExp, err = exporter.NewTraceExporter(ctx, exporterCfg)
	}
	if err != nil || traceExp == nil {
		return nil, cleanup, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	if retries > 0 {
		metricExp, err = exporter.NewMetricExporterWithRetry(ctx, exporterCfg, retries)
	} else {
		metricExp, err = exporter.NewMetricExporter(ctx, exporterCfg)
	}
	if err != nil || metricExp == nil {
		return nil, cleanup, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	if retries > 0 {
		logExp, err = exporter.NewLogExporterWithRetry(ctx, exporterCfg, retries)
	} else {
		logExp, err = exporter.NewLogExporter(ctx, exporterCfg)
	}
	if err != nil || logExp == nil {
		return nil, cleanup, fmt.Errorf("failed to create log exporter: %w", err)
	}

	providers := &provider.Providers{
		TracerProvider: provider.NewTracerProvider(ctx, traceExp, res),
		MeterProvider:  provider.NewMeterProvider(ctx, metricExp, res),
		LoggerProvider: provider.NewLoggerProvider(ctx, logExp, res),
	}

	return providers, cleanup, nil
}

func (t *Telemetry) Flush(ctx context.Context) error {
	if t == nil || t.providers == nil {
		return nil
	}
	start := time.Now()
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.shutdownTimeout)
		defer cancel()
	}
	err := t.providers.ForceFlush(ctx)
	if duration := time.Since(start); duration > t.shutdownTimeout/2 && t.logger != nil {
		t.logger.Warn("slow telemetry flush", "duration_ms", duration.Milliseconds(), "timeout_ms", t.shutdownTimeout.Milliseconds())
	}
	return err
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil || t.providers == nil {
		return nil
	}
	t.shutdownOnce.Do(func() {
		defer close(t.shutdownDone)
		start := time.Now()
		if t.evictionCancel != nil {
			t.evictionCancel()
			t.evictionDone.Wait()
		}
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, t.shutdownTimeout)
			defer cancel()
		}
		t.shutdownErr = t.providers.Shutdown(ctx)
		if duration := time.Since(start); duration > t.shutdownTimeout/2 && t.logger != nil {
			t.logger.Warn("slow telemetry shutdown", "duration_ms", duration.Milliseconds(), "timeout_ms", t.shutdownTimeout.Milliseconds())
		}
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
	return MetricAPI{
		meter:        t.meter,
		logger:       t.logger,
		counterCache: &t.counterCache,
		gaugeCache:   &t.gaugeCache,
		histoCache:   &t.histoCache,
		cacheSize:    &t.metricCacheSize,
	}
}

// Attr returns the unified attribute builder for use across all telemetry types.
// Attributes created with this API work for tracing, metrics, and logging.
//
// Example:
//
//	attrs := []attribute.KeyValue{
//	    tel.Attr().String("user.id", "123"),
//	    tel.Attr().Int("status", 200),
//	}
//	tel.Trace().SetAttributes(ctx, attrs...)
//	tel.Metric().Counter("requests").Add(ctx, 1, metric.WithAttributes(attrs...))
func (t *Telemetry) Attr() AttributeAPI {
	return AttributeAPI{}
}

// MeasureDuration executes fn and records its duration to a histogram metric.
// If fn returns an error, it's automatically recorded in the current span (if one exists).
//
// Example:
//
//	err := tel.MeasureDuration(ctx, "db.query.duration", func() error {
//	    return db.Query(ctx, ...)
//	})
func (t *Telemetry) MeasureDuration(ctx context.Context, metricName string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	t.Metric().RecordDuration(ctx, metricName, duration)

	if err != nil {
		t.Trace().RecordError(ctx, err)
	}

	return err
}

// TraceFunction creates a span, executes fn with the span context, and ends the span.
// Errors returned by fn are automatically recorded in the span.
//
// Example:
//
//	err := tel.TraceFunction(ctx, "process.order", func(ctx context.Context) error {
//	    tel.Log().Info(ctx, "processing order")
//	    return processOrder(ctx, orderID)
//	})
func (t *Telemetry) TraceFunction(ctx context.Context, spanName string, fn func(context.Context) error) error {
	newCtx, stop := t.Trace().StartSpan(ctx, spanName)
	defer stop()

	err := fn(newCtx)
	if err != nil {
		t.Trace().RecordError(newCtx, err)
	}

	return err
}

// WithSpan creates a span, executes fn with the span context, and ends the span.
// This is an alias for TraceFunction for more ergonomic usage.
//
// Example:
//
//	err := tel.WithSpan(ctx, "database.query", func(ctx context.Context) error {
//	    return db.Query(ctx, query)
//	})
func (t *Telemetry) WithSpan(ctx context.Context, spanName string, fn func(context.Context) error) error {
	return t.TraceFunction(ctx, spanName, fn)
}

func (t *Telemetry) TracerProvider() *sdktrace.TracerProvider {
	if t == nil || t.providers == nil {
		return nil
	}
	return t.providers.TracerProvider
}

func (t *Telemetry) MeterProvider() *sdkmetric.MeterProvider {
	if t == nil || t.providers == nil {
		return nil
	}
	return t.providers.MeterProvider
}

func (t *Telemetry) LoggerProvider() *sdklog.LoggerProvider {
	if t == nil || t.providers == nil {
		return nil
	}
	return t.providers.LoggerProvider
}

func (t *Telemetry) startMetricEviction() {
	t.evictionDone.Add(1)
	go func() {
		defer t.evictionDone.Done()
		ticker := time.NewTicker(metricEvictionInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				t.evictStaleMetrics()
			case <-t.evictionCtx.Done():
				return
			}
		}
	}()
}

func (t *Telemetry) evictStaleMetrics() {
	cutoff := time.Now().UnixNano() - int64(metricTTL)
	evicted := 0

	evictFromCache := func(cache *sync.Map) {
		cache.Range(func(key, value any) bool {
			if entry, ok := value.(*metricCacheEntry); ok && entry.lastUsed.Load() < cutoff {
				cache.Delete(key)
				evicted++
			}
			return true
		})
	}

	evictFromCache(&t.counterCache)
	evictFromCache(&t.histoCache)
	evictFromCache(&t.gaugeCache)

	if evicted > 0 {
		t.metricCacheSize.Add(-int64(evicted))
		if t.logger != nil {
			t.logger.Debug("evicted stale metrics", "evicted", evicted, "remaining", t.metricCacheSize.Load())
		}
	}
}
