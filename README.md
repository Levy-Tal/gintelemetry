# gintelemetry

[![Go Reference](https://pkg.go.dev/badge/github.com/Levy-Tal/gintelemetry.svg)](https://pkg.go.dev/github.com/Levy-Tal/gintelemetry)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**Simple OpenTelemetry for Gin.** One function call. Zero boilerplate. Full observability.

## Why?

Setting up OpenTelemetry is tedious. This package does it for you.

```go
tel, router, _ := gintelemetry.Start(ctx, gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "localhost:4317",
    Insecure:    true,
})
defer tel.Shutdown(ctx)
```

That's it. You now have traces, metrics, and logs flowing to your collector.

## Philosophy

**gintelemetry** is a thin, focused wrapper around OpenTelemetry's Go SDK for Gin applications:

- ✅ **Minimal abstraction** - Stays close to OTEL patterns
- ✅ **No magic** - Explicit, predictable behavior
- ✅ **Leverage OTEL SDK** - Uses SDK features directly
- ✅ **Simple to use** - Easy onboarding
- ✅ **Easy to extend** - Build your own helpers on top

## Install

```bash
go get github.com/Levy-Tal/gintelemetry@v2
```

## Quick Start

```go
package main

import (
    "context"
    "github.com/Levy-Tal/gintelemetry"
)

func main() {
    ctx := context.Background()
    
    tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
        ServiceName: "my-api",
        Endpoint:    "localhost:4317",
        Insecure:    true,
        LogLevel:    gintelemetry.LevelInfo,
    })
    if err != nil {
        panic(err)
    }
    defer tel.Shutdown(ctx)

    router.GET("/hello", func(c *gin.Context) {
        ctx := c.Request.Context()
        
        // Logging with automatic trace correlation
        tel.Log().Info(ctx, "handling request")
        
        // Metrics
        tel.Metric().Counter("requests").Add(ctx, 1,
            metric.WithAttributes(tel.Attr().String("endpoint", "hello")),
        )
        
        c.String(200, "Hello!")
    })

    router.Run(":8080")
}
```

## Usage

### Configuration

**Basic (gRPC, default):**

```go
config := gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "localhost:4317",
    Insecure:    true,
    LogLevel:    gintelemetry.LevelDebug,
}
```

**HTTP Protocol:**

```go
config := gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "localhost:4318",
    Protocol:    gintelemetry.ProtocolHTTP,
    Insecure:    true,
}
```

**With Global Attributes:**

```go
config := gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "localhost:4317",
    Insecure:    true,
    GlobalAttributes: map[string]string{
        "environment": "production",
        "region":      "us-east-1",
    },
}
```

**From Environment Variables:**

Set standard OpenTelemetry environment variables:

- `OTEL_SERVICE_NAME` - Service name
- `OTEL_EXPORTER_OTLP_ENDPOINT` - Collector endpoint

```go
// Reads from environment variables
config := gintelemetry.Config{}
tel, router, err := gintelemetry.Start(ctx, config)
```

### Logging

Logs are sent to both OTLP collector and stdout for easy development.

**Simple Logging:**

```go
tel.Log().Info(ctx, "user logged in", "user_id", 123)
tel.Log().Warn(ctx, "rate limit approaching", "current", 95, "limit", 100)
tel.Log().Error(ctx, "database connection failed", "error", err.Error())
tel.Log().Debug(ctx, "cache hit", "key", cacheKey)
```

**With Structured Fields:**

```go
tel.Log().Info(ctx, "order created",
    "order_id", order.ID,
    "user_id", user.ID,
    "amount", order.Total,
)
```

### Metrics

**Counters:**

```go
counter := tel.Metric().Counter("requests")
counter.Add(ctx, 1, metric.WithAttributes(
    tel.Attr().String("method", "GET"),
    tel.Attr().String("endpoint", "/api/users"),
))
```

**Histograms:**

```go
histogram := tel.Metric().Histogram("request_duration_ms")
histogram.Record(ctx, duration.Milliseconds())
```

**Gauges:**

```go
gauge := tel.Metric().Gauge("active_connections")
gauge.Record(ctx, 42)
```

**Float64 Metrics:**

```go
tel.Metric().Float64Counter("cpu_time_seconds").Add(ctx, 0.523)
tel.Metric().Float64Histogram("response_time_seconds").Record(ctx, 0.142)
tel.Metric().Float64Gauge("cpu_usage_percent").Record(ctx, 45.2)
```

### Tracing

**Manual Spans:**

```go
func processOrder(ctx context.Context, tel *gintelemetry.Telemetry, order Order) error {
    ctx, stop := tel.Trace().StartSpan(ctx, "process_order")
    defer stop()
    
    // Add attributes
    tel.Trace().SetAttributes(ctx,
        tel.Attr().String("order.id", order.ID),
        tel.Attr().Int("order.items", len(order.Items)),
    )
    
    // Your business logic
    if err := validateOrder(order); err != nil {
        tel.Trace().RecordError(ctx, err)
        return err
    }
    
    return nil
}
```

**With Span Options:**

```go
ctx, stop := tel.Trace().StartSpan(ctx, "database.query",
    trace.WithAttributes(
        tel.Attr().String("db.system", "postgres"),
        tel.Attr().String("db.operation", "SELECT"),
    ),
)
defer stop()
```

**Add Events:**

```go
tel.Trace().AddEvent(ctx, "cache.miss",
    tel.Attr().String("cache.key", key),
)
```

**Set Status:**

```go
tel.Trace().SetStatus(ctx, gintelemetry.StatusOK, "operation completed")
```

### Attributes

**Common Types:**

```go
tel.Attr().String("user.id", "abc123")
tel.Attr().Int("http.status_code", 200)
tel.Attr().Int64("bytes_sent", 1024)
tel.Attr().Float64("cpu.usage", 45.2)
tel.Attr().Bool("cache.hit", true)
```

## Building Custom Helpers

The simplified API makes it easy to build your own helpers for common patterns:

### Example: WithSpan Helper

```go
// WithSpan executes a function within a span
func WithSpan(tel *gintelemetry.Telemetry, ctx context.Context, name string, fn func(context.Context) error) error {
    ctx, stop := tel.Trace().StartSpan(ctx, name)
    defer stop()
    
    err := fn(ctx)
    if err != nil {
        tel.Trace().RecordError(ctx, err)
    }
    
    return err
}

// Usage
err := WithSpan(tel, ctx, "process_payment", func(ctx context.Context) error {
    return processPayment(ctx, payment)
})
```

### Example: MeasureDuration Helper

```go
// MeasureDuration measures and records the duration of a function
func MeasureDuration(tel *gintelemetry.Telemetry, ctx context.Context, metricName string, fn func() error) error {
    start := time.Now()
    err := fn()
    
    histogram := tel.Metric().Histogram(metricName)
    histogram.Record(ctx, time.Since(start).Milliseconds())
    
    if err != nil {
        tel.Trace().RecordError(ctx, err)
    }
    
    return err
}

// Usage
err := MeasureDuration(tel, ctx, "db.query.duration", func() error {
    return db.Query(ctx, query)
})
```

### Example: Background Job Helper

```go
// WithBackgroundJob wraps a background job with tracing, logging, and metrics
func WithBackgroundJob(tel *gintelemetry.Telemetry, ctx context.Context, name string, fn func(context.Context) error) error {
    start := time.Now()
    
    // Create a new root span for the background job
    ctx, stop := tel.Trace().StartSpan(ctx, name,
        trace.WithSpanKind(trace.SpanKindInternal),
        trace.WithAttributes(tel.Attr().String("job.type", "background")),
    )
    defer stop()
    
    tel.Log().Info(ctx, "job started", "job", name)
    
    err := fn(ctx)
    
    duration := time.Since(start)
    histogram := tel.Metric().Histogram("job.duration")
    histogram.Record(ctx, duration.Milliseconds(),
        metric.WithAttributes(
            tel.Attr().String("job", name),
            tel.Attr().Bool("success", err == nil),
        ),
    )
    
    if err != nil {
        tel.Trace().RecordError(ctx, err)
        tel.Log().Error(ctx, "job failed", "job", name, "error", err.Error())
        
        counter := tel.Metric().Counter("job.failures")
        counter.Add(ctx, 1, metric.WithAttributes(tel.Attr().String("job", name)))
    } else {
        tel.Log().Info(ctx, "job completed", "job", name)
        
        counter := tel.Metric().Counter("job.completions")
        counter.Add(ctx, 1, metric.WithAttributes(tel.Attr().String("job", name)))
    }
    
    return err
}

// Usage
err := WithBackgroundJob(tel, context.Background(), "scraper.metrics", func(ctx context.Context) error {
    return scrapeMetrics(ctx)
})
```

## Testing

Use `NewTestConfig` for tests:

```go
func TestHandler(t *testing.T) {
    ctx := context.Background()
    tel, router, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-service"))
    if err != nil {
        t.Fatalf("failed to start telemetry: %v", err)
    }
    defer tel.Shutdown(ctx)

    router.GET("/test", myHandler(tel))
    
    // Make requests...
}
```

## API Reference

### Core

| Method | Description |
|--------|-------------|
| `Start(ctx, config)` | Initialize telemetry and return Telemetry instance + Gin router |
| `Shutdown(ctx)` | Gracefully shutdown telemetry |
| `Flush(ctx)` | Force flush telemetry data |

### Telemetry

| Method | Description |
|--------|-------------|
| `Log()` | Get logging API |
| `Metric()` | Get metrics API |
| `Trace()` | Get tracing API |
| `Attr()` | Get attribute helpers |
| `TracerProvider()` | Get underlying tracer provider |
| `MeterProvider()` | Get underlying meter provider |
| `LoggerProvider()` | Get underlying logger provider |

### LogAPI

| Method | Description |
|--------|-------------|
| `Debug(ctx, msg, attrs...)` | Log debug message |
| `Info(ctx, msg, attrs...)` | Log info message |
| `Warn(ctx, msg, attrs...)` | Log warning message |
| `Error(ctx, msg, attrs...)` | Log error message |
| `Logger()` | Get underlying slog.Logger |
| `With(attrs...)` | Create logger with attributes |
| `WithGroup(name)` | Create logger with group |

### MetricAPI

| Method | Description |
|--------|-------------|
| `Counter(name, opts...)` | Get Int64Counter instrument |
| `Histogram(name, opts...)` | Get Int64Histogram instrument |
| `Gauge(name, opts...)` | Get Int64Gauge instrument |
| `Float64Counter(name, opts...)` | Get Float64Counter instrument |
| `Float64Histogram(name, opts...)` | Get Float64Histogram instrument |
| `Float64Gauge(name, opts...)` | Get Float64Gauge instrument |

### TraceAPI

| Method | Description |
|--------|-------------|
| `StartSpan(ctx, name, opts...)` | Start a span and return (ctx, stop func) |
| `SetAttributes(ctx, attrs...)` | Set attributes on current span |
| `AddEvent(ctx, name, attrs...)` | Add event to current span |
| `RecordError(ctx, err)` | Record error in current span |
| `SetStatus(ctx, code, desc)` | Set status of current span |
| `SpanFromContext(ctx)` | Get current span from context |

### AttributeAPI

| Method | Description |
|--------|-------------|
| `String(key, value)` | Create string attribute |
| `Int(key, value)` | Create int attribute |
| `Int64(key, value)` | Create int64 attribute |
| `Float64(key, value)` | Create float64 attribute |
| `Bool(key, value)` | Create bool attribute |
| `Strings(key, values)` | Create string slice attribute |

## Examples

See the [examples](examples/) directory for complete working examples:

- [examples/basic](examples/basic/) - Simple HTTP API
- [examples/worker](examples/worker/) - Background jobs and workers with custom helpers

## License

Apache 2.0. See [LICENSE](LICENSE).
