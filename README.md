# gintelemetry

[![Go Reference](https://pkg.go.dev/badge/github.com/Levy-Tal/gintelemetry.svg)](https://pkg.go.dev/github.com/Levy-Tal/gintelemetry)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**Opinionated OpenTelemetry for Gin.** One function call. Zero boilerplate. Full observability.

## Why?

Setting up OpenTelemetry is tedious. This package does it for you.

```go
router, shutdown, _ := gintelemetry.Start(ctx, gintelemetry.Config{
    ServiceName: "my-service",
})
defer shutdown(ctx)
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
    
    router, shutdown, err := gintelemetry.Start(ctx, gintelemetry.Config{
        ServiceName: "my-api",
        LogLevel:    gintelemetry.LevelInfo,  // Optional, defaults to LevelInfo
    })
    if err != nil {
        panic(err)
    }
    defer shutdown(ctx)

    router.GET("/hello", func(c *gin.Context) {
        ctx := c.Request.Context()
        
        // Logging (with trace correlation)
        gintelemetry.Log.Info(ctx, "handling request")
        
        // Metrics
        gintelemetry.Metric.Counter("requests").Add(ctx, 1,
            gintelemetry.Metric.String("endpoint", "hello"),
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
```

### Logging

```go
ctx := c.Request.Context()

gintelemetry.Log.Info(ctx, "message", "key", "value")
gintelemetry.Log.Warn(ctx, "warning")
gintelemetry.Log.Error(ctx, "error", "error", err.Error())
gintelemetry.Log.Debug(ctx, "debug info")
```

### Tracing

```go
// Automatic tracing for all routes (via otelgin middleware)

// Manual spans
ctx, stop := gintelemetry.Trace.StartSpan(ctx, "operation")
defer stop()

// With attributes
ctx, stop := gintelemetry.Trace.StartSpanWithAttributes(ctx, "db.query",
    gintelemetry.Trace.String("db.system", "postgres"),
    gintelemetry.Trace.Int("rows", 42),
)
defer stop()

// Add attributes
gintelemetry.Trace.SetAttributes(ctx,
    gintelemetry.Trace.String("user.id", userId),
)

// Record errors
if err != nil {
    gintelemetry.Trace.RecordError(ctx, err)
}
```

### Metrics

```go
// Counter
gintelemetry.Metric.Counter("requests.total").Add(ctx, 1,
    gintelemetry.Metric.String("method", "GET"),
)

// Histogram
gintelemetry.Metric.Histogram("request.duration.ms").Record(ctx, duration,
    gintelemetry.Metric.String("endpoint", "/api"),
)

// Gauge
gintelemetry.Metric.Gauge("memory.bytes").Record(ctx, memUsed)

// Float64 variants available
gintelemetry.Metric.Float64Counter("cpu.percent").Add(ctx, 0.75)
gintelemetry.Metric.Float64Histogram("size.mb").Record(ctx, 2.5)
gintelemetry.Metric.Float64Gauge("temperature").Record(ctx, 23.5)
```

### Attributes

Each namespace has its own attribute helpers:

```go
// For tracing
gintelemetry.Trace.String("key", "value")
gintelemetry.Trace.Int("count", 42)
gintelemetry.Trace.Bool("success", true)
gintelemetry.Trace.Float64("ratio", 0.95)

// For metrics
gintelemetry.Metric.String("endpoint", "/api")
gintelemetry.Metric.Int("status", 200)
```

## Features

- ✅ **One-line setup** - `Start()` does everything
- ✅ **Auto-instrumentation** - All routes traced automatically
- ✅ **Trace correlation** - Logs include trace/span IDs
- ✅ **Clean API** - `Log.*`, `Metric.*`, `Trace.*` namespaces
- ✅ **Zero imports** - Only import `gintelemetry` (and `gin`)
- ✅ **Flexible transport** - Support for gRPC and HTTP protocols
- ✅ **Secure** - Support for TLS and mTLS
- ✅ **Global attributes** - Add team, environment, region to all telemetry
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
| **Logging** | `Log.Info/Warn/Error/Debug(ctx, msg, ...)` | Structured logging |
| **Tracing** | `Trace.StartSpan(ctx, name)` | Create spans |
| | `Trace.SetAttributes(ctx, ...)` | Add span attributes |
| | `Trace.RecordError(ctx, err)` | Record errors |
| **Metrics** | `Metric.Counter(name)` | Monotonic counter |
| | `Metric.Histogram(name)` | Value distribution |
| | `Metric.Gauge(name)` | Point-in-time value |

## Examples

See the [examples](examples/) directory for complete working examples.

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
