# gintelemetry

[![Go Reference](https://pkg.go.dev/badge/github.com/Levy-Tal/gintelemetry.svg)](https://pkg.go.dev/github.com/Levy-Tal/gintelemetry)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**Opinionated OpenTelemetry for Gin.** One function call. Zero boilerplate. Full observability.

## Why?

Setting up OpenTelemetry is tedious. This package does it for you.

```go
tel, router, _ := gintelemetry.Start(ctx, gintelemetry.Config{
    ServiceName: "my-service",
})
defer tel.Shutdown(ctx)
```

That's it. You now have traces, metrics, and logs flowing to your collector.

## Install

```bash
go get github.com/Levy-Tal/gintelemetry
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
        tel.Metric().IncrementCounter(ctx, "requests",
            tel.Attr().String("endpoint", "hello"),
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
    Endpoint:    "localhost:4317",          // gRPC default port
    LogLevel:    gintelemetry.LevelDebug,
}
```

**HTTP Protocol:**

```go
config := gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "localhost:4318",          // HTTP default port
}.WithHTTP()

// Or explicitly
config.Protocol = gintelemetry.ProtocolHTTP
```

**With TLS (Trusted CA):**

```go
config := gintelemetry.Config{
    ServiceName: "my-service",
}.WithTrustedCA("/path/to/ca.crt")
```

**With mTLS (Mutual TLS):**

```go
config := gintelemetry.Config{
    ServiceName: "my-service",
}.WithMTLS(
    "/path/to/client.crt",
    "/path/to/client.key",
    "/path/to/ca.crt",
)
```

**HTTP with TLS:**

```go
config := gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "localhost:4318",
}.WithHTTP().WithTrustedCA("/path/to/ca.crt")
```

**With Global Attributes:**

```go
// Using map
config := gintelemetry.Config{
    ServiceName: "my-service",
    GlobalAttributes: map[string]string{
        "team":        "platform",
        "environment": "production",
        "region":      "us-east-1",
    },
}

// Using builder pattern
config := gintelemetry.Config{
    ServiceName: "my-service",
}.WithGlobalAttribute("team", "platform").
  WithGlobalAttribute("environment", "production").
  WithGlobalAttribute("region", "us-east-1")

// Or set all at once
config := gintelemetry.Config{
    ServiceName: "my-service",
}.WithGlobalAttributes(map[string]string{
    "team":        "platform",
    "environment": "production",
})

// Or use environment variables (no recompilation needed!)
// export OTEL_RESOURCE_ATTRIBUTES="team=platform,environment=production,region=us-east-1"
config := gintelemetry.Config{
    ServiceName: "my-service",
    // Attributes will be loaded from OTEL_RESOURCE_ATTRIBUTES
}
```

### Logging

```go
ctx := c.Request.Context()

tel.Log().Info(ctx, "message", "key", "value")
tel.Log().Warn(ctx, "warning")
tel.Log().Error(ctx, "error", "error", err.Error())
tel.Log().Debug(ctx, "debug info")
```

### Tracing

```go
// Automatic tracing for all routes (via otelgin middleware)

// Manual spans
ctx, stop := tel.Trace().StartSpan(ctx, "operation")
defer stop()

// With attributes
ctx, stop := tel.Trace().StartSpanWithAttributes(ctx, "db.query",
    tel.Attr().String("db.system", "postgres"),
    tel.Attr().Int("rows", 42),
)
defer stop()

// Add attributes
tel.Trace().SetAttributes(ctx,
    tel.Attr().String("user.id", userId),
)

// Record errors
if err != nil {
    tel.Trace().RecordError(ctx, err)
}
```

### Metrics

**Convenience Methods (Recommended):**

```go
// Counter - increment by 1
tel.Metric().IncrementCounter(ctx, "requests.total",
    tel.Attr().String("method", "GET"),
)

// Counter - add specific value
tel.Metric().AddCounter(ctx, "bytes.sent", bytesCount)

// Histogram - record value
tel.Metric().RecordHistogram(ctx, "request.duration.ms", duration)

// Duration - automatically converts to milliseconds
tel.Metric().RecordDuration(ctx, "db.query.duration", time.Since(start))

// Gauge
tel.Metric().RecordGauge(ctx, "memory.bytes", memUsed)

// Float64 variants
tel.Metric().AddFloat64Counter(ctx, "cpu.percent", 0.75)
tel.Metric().RecordFloat64Histogram(ctx, "size.mb", 2.5)
tel.Metric().RecordFloat64Gauge(ctx, "temperature", 23.5)
```

**Direct Instrument Access (Advanced):**

```go
// For more control, get instruments directly
tel.Metric().Counter("requests.total").Add(ctx, 1,
    metric.WithAttributes(tel.Attr().String("method", "GET")),
)

tel.Metric().Histogram("request.duration.ms").Record(ctx, duration,
    metric.WithAttributes(tel.Attr().String("endpoint", "/api")),
)
```

### Attributes

Use `tel.Attr()` to create attributes that work everywhere:

```go
attrs := []attribute.KeyValue{
    tel.Attr().String("user.id", "123"),
    tel.Attr().Int("status", 200),
}

// Works for tracing
tel.Trace().SetAttributes(ctx, attrs...)

// Works for metrics
tel.Metric().IncrementCounter(ctx, "requests", attrs...)

// Available attribute types
tel.Attr().String(key, value)
tel.Attr().Int(key, value)
tel.Attr().Int64(key, value)
tel.Attr().Float64(key, value)
tel.Attr().Bool(key, value)
tel.Attr().Strings(key, []string{...})
```

### Convenience Helpers

**Measure Duration:**

```go
// Automatically records duration and handles errors
err := tel.MeasureDuration(ctx, "db.query.duration", func() error {
    return db.Query(ctx, query)
})
```

**Trace Functions:**

```go
// Automatically creates span and records errors
err := tel.WithSpan(ctx, "process.order", func(ctx context.Context) error {
    tel.Log().Info(ctx, "processing order")
    return processOrder(ctx, orderID)
})
```

## Understanding Context

All telemetry methods accept a `context.Context` for **trace correlation**. This allows logs, metrics, and spans to be linked together.

### How It Works

```go
// 1. Start with request context (automatically has a span from otelgin middleware)
ctx := c.Request.Context()

// 2. Log with context - automatically includes trace/span IDs
tel.Log().Info(ctx, "processing request")

// 3. Create child spans
ctx, stop := tel.Trace().StartSpan(ctx, "database.query")
defer stop()

// 4. All operations use the same context chain
tel.Log().Info(ctx, "querying database")  // Linked to "database.query" span
tel.Metric().RecordDuration(ctx, "db.latency", duration)
```

### What If Context Has No Span?

Using `context.Background()` or a context without a span is **safe** but:

- ✅ Logs will still be recorded
- ✅ Metrics will still be recorded
- ✅ Operations will not fail or panic
- ❌ Trace correlation will be lost (logs won't link to traces)

**Best Practice:** Always propagate context from the request through your call chain.

## Error Handling

gintelemetry is designed to **never fail your application**. All telemetry operations are safe and handle errors gracefully.

### Silent Failures

If telemetry operations fail (e.g., logger is nil, span not recording, collector unavailable), they:

- ✅ Return without error
- ✅ Never panic
- ✅ Don't block your application
- ⚠️ May log warnings (if logger is available)

```go
// These are all safe, even if telemetry isn't initialized properly
tel.Log().Info(ctx, "message")          // No-op if logger is nil
tel.Trace().RecordError(ctx, err)       // No-op if span not recording
tel.Metric().IncrementCounter(ctx, "x") // Falls back to no-op if creation fails
```

### When to Check Errors

The only error you should handle is from `Start()` and `Shutdown()`:

```go
tel, router, err := gintelemetry.Start(ctx, config)
if err != nil {
    // This is a real startup error - handle it
    log.Fatalf("failed to initialize telemetry: %v", err)
}
defer tel.Shutdown(ctx)
```

## Best Practices

### Metric Naming

**❌ BAD - Dynamic metric names:**

```go
// Creates a new cached metric for EVERY user!
tel.Metric().Counter("user_" + userID + "_requests").Add(ctx, 1)
```

**✅ GOOD - Use attributes:**

```go
// Single metric with user.id attribute
tel.Metric().IncrementCounter(ctx, "user.requests",
    tel.Attr().String("user.id", userID),
)
```

### Why?

Metrics are cached (up to 10,000). Dynamic names exhaust the cache and hurt performance.

### Context Propagation

**✅ GOOD:**

```go
func handleRequest(c *gin.Context) {
    ctx := c.Request.Context()
    
    // Pass context through call chain
    processOrder(ctx, orderID)
}

func processOrder(ctx context.Context, orderID string) error {
    tel.Log().Info(ctx, "processing order", "order_id", orderID)
    return nil
}
```

**❌ BAD:**

```go
func processOrder(orderID string) error {
    // Lost context - no trace correlation!
    tel.Log().Info(context.Background(), "processing order")
    return nil
}
```

## Testing

Use `NewTestConfig()` for easy testing without requiring a real collector:

```go
func TestMyHandler(t *testing.T) {
    ctx := context.Background()
    
    // Creates config with no-op exporters
    tel, router, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-service"))
    if err != nil {
        t.Fatalf("failed to start telemetry: %v", err)
    }
    defer tel.Shutdown(ctx)
    
    // Test your handlers
    router.GET("/test", myHandler)
    
    // Make test requests...
}
```

The test config:

- Uses insecure localhost connection
- Sets log level to Error (less noise)
- Disables retries (fails fast)
- Safe to use even without a collector running

## Environment Variables

gintelemetry supports standard OpenTelemetry environment variables:

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `OTEL_SERVICE_NAME` | Service name | Required (or set in Config) | `my-api` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector endpoint | Required (or set in Config) | `localhost:4317` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | Protocol (grpc/http) | grpc | `grpc` or `http` |
| `OTEL_RESOURCE_ATTRIBUTES` | Global attributes | None | `team=platform,env=prod` |

**Example:**

```bash
export OTEL_SERVICE_NAME=my-api
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
export OTEL_RESOURCE_ATTRIBUTES="team=platform,environment=production,region=us-east-1"
go run main.go
```

```go
// Config fields take precedence over env vars
tel, router, _ := gintelemetry.Start(ctx, gintelemetry.Config{
    // ServiceName can come from OTEL_SERVICE_NAME env var
    // Endpoint can come from OTEL_EXPORTER_OTLP_ENDPOINT env var
    // Global attributes can come from OTEL_RESOURCE_ATTRIBUTES env var
})
```

### Global Attributes from Environment

The `OTEL_RESOURCE_ATTRIBUTES` environment variable allows you to add global attributes without recompiling:

```bash
# Development
export OTEL_RESOURCE_ATTRIBUTES="team=backend,environment=dev,region=local"

# Production
export OTEL_RESOURCE_ATTRIBUTES="team=backend,environment=production,region=us-east-1,version=1.2.3"
```

**Format:** Comma-separated `key=value` pairs

**Precedence:** Config attributes override environment attributes (if both specify the same key)

**Use Case:** Change team, environment, region, version without rebuilding - perfect for:

- Different deployment environments
- Multi-tenant deployments
- CI/CD pipelines
- Container orchestration (Kubernetes, Docker)

## Features

- ✅ **One-line setup** - `Start()` does everything
- ✅ **Auto-instrumentation** - All routes traced automatically
- ✅ **Trace correlation** - Logs include trace/span IDs
- ✅ **Clean API** - `Log.*`, `Metric.*`, `Trace.*` namespaces
- ✅ **Convenience methods** - `IncrementCounter`, `RecordDuration`, `WithSpan`
- ✅ **Unified attributes** - `Attr()` works across all telemetry types
- ✅ **Zero imports** - Only import `gintelemetry` (and `gin`)
- ✅ **Flexible transport** - Support for gRPC and HTTP protocols
- ✅ **Secure** - Support for TLS and mTLS
- ✅ **Global attributes** - Add team, environment, region to all telemetry
- ✅ **Testing utilities** - `NewTestConfig()` for easy testing
- ✅ **Production-ready** - Batching, resource attribution, graceful shutdown

## OpenTelemetry Collector

You need an OTLP collector running. Quick start with Docker:

```bash
docker run -d --name otel-collector \
  -p 4317:4317 \
  -p 4318:4318 \
  otel/opentelemetry-collector:latest
```

Or use Jaeger all-in-one:

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/all-in-one:latest
```

Then visit <http://localhost:16686> to view traces.

## What You Get

**Before (manual OpenTelemetry):**

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
    // ... 10+ more imports
    // ... 100+ lines of setup code
)
```

**After (gintelemetry):**

```go
import "github.com/Levy-Tal/gintelemetry"

router, shutdown, _ := gintelemetry.Start(ctx, config)
defer shutdown(ctx)
// Done!
```

## API Reference

| Category | Function | Description |
|----------|----------|-------------|
| **Setup** | `Start(ctx, config)` | Initialize everything |
| **Attributes** | `Attr().String/Int/Bool(...)` | Unified attributes for all telemetry |
| **Logging** | `Log().Info/Warn/Error/Debug(ctx, msg, ...)` | Structured logging |
| **Tracing** | `Trace().StartSpan(ctx, name)` | Create spans |
| | `Trace().SetAttributes(ctx, ...)` | Add span attributes |
| | `Trace().RecordError(ctx, err)` | Record errors |
| | `WithSpan(ctx, name, fn)` | Execute function in span |
| **Metrics** | `Metric().IncrementCounter(ctx, name, ...)` | Increment counter |
| | `Metric().RecordDuration(ctx, name, duration, ...)` | Record timing |
| | `Metric().RecordHistogram(ctx, name, value, ...)` | Record value |
| | `Metric().RecordGauge(ctx, name, value, ...)` | Record gauge |
| | `Metric().Counter/Histogram/Gauge(name)` | Get instrument directly |
| **Helpers** | `MeasureDuration(ctx, name, fn)` | Time and trace function |
| **Testing** | `NewTestConfig(serviceName)` | Test configuration |

## Examples

Complete working examples with detailed explanations:

### [Basic Example](examples/basic/)

Simple hello world showing core features:

- Basic setup and configuration
- Logging with trace correlation
- Counter metrics with attributes
- Manual span creation
- Convenience helpers

### [Database Example](examples/database/)

Database instrumentation patterns:

- Query tracing with timing
- Database operation metrics
- Error handling and recording
- Connection pool monitoring
- Best practices for DB instrumentation

### [Testing Example](examples/testing/)

How to test instrumented code:

- Using `NewTestConfig()` for tests
- Testing services with telemetry
- HTTP handler testing
- Concurrent request testing
- Benchmarking instrumented code

### [Environment Variables Example](examples/env-vars/)

Configure everything via environment variables:

- Zero hardcoded configuration
- Global attributes from `OTEL_RESOURCE_ATTRIBUTES`
- Same binary for all environments
- Docker and Kubernetes examples
- CI/CD friendly configuration

Each example includes:

- Complete runnable code
- Detailed README with explanations
- Docker Compose for local collector
- Common patterns and best practices

**Quick Start:**

```bash
cd examples/basic
docker compose up -d  # Start Jaeger
go run main.go        # Run example
# Visit http://localhost:16686 to view traces
```

## Documentation

- [API Reference](https://pkg.go.dev/github.com/Levy-Tal/gintelemetry)
- [Architecture](docs/ARCHITECTURE.md)
- [Contributing](docs/CONTRIBUTING.md)
- [Changelog](docs/CHANGELOG.md)

## Requirements

- Go 1.24+
- OpenTelemetry Collector (or Jaeger/Grafana/etc.)

## License

Apache-2.0
