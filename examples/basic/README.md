# Basic Example

A simple hello world example demonstrating core gintelemetry features.

## What This Example Shows

- Basic telemetry setup with `Start()`
- Automatic request tracing via middleware
- Logging with trace correlation
- Counter metrics with attributes
- Manual span creation
- Error recording in spans
- Convenience helpers (`MeasureDuration`, `WithSpan`)
- Unified attribute API

## Prerequisites

You need an OpenTelemetry collector running. The easiest way is with Docker:

### Using Jaeger All-in-One (includes collector and UI)

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/all-in-one:latest
```

Then visit <http://localhost:16686> to view traces.

### Using OpenTelemetry Collector

```bash
docker run -d --name otel-collector \
  -p 4317:4317 \
  -p 4318:4318 \
  otel/opentelemetry-collector:latest
```

## Running the Example

```bash
cd examples/basic
go run main.go
```

The server will start on port 8080.

## Try It Out

### Hello Endpoint

```bash
curl http://localhost:8080/hello
```

This demonstrates:

- Basic logging
- Counter metrics
- Automatic trace from middleware

### Process Endpoint

```bash
curl http://localhost:8080/process/123
```

This demonstrates:

- Manual span creation with attributes
- Nested spans (parent -> child)
- Error handling and recording
- Log/trace correlation

### Measure Endpoint

```bash
curl http://localhost:8080/measure
```

This demonstrates:

- `MeasureDuration` convenience helper
- Automatic duration recording

## View the Results

1. **Traces**: Open <http://localhost:16686> (if using Jaeger)
2. **Service**: Select "basic-example" from the dropdown
3. **Find Traces**: Click "Find Traces"
4. **Explore**: Click on any trace to see spans, logs, and attributes

You'll see:

- Request traces with all spans
- Span attributes (endpoint, method, item.id, etc.)
- Timing information
- Error details (if you trigger an error)

## What to Notice

1. **Trace Correlation**: Logs include trace/span IDs automatically
2. **Nested Spans**: The `process/:id` endpoint creates a parent span, then a child "database.query" span
3. **Attributes**: All spans include relevant attributes (ids, endpoints, etc.)
4. **Metrics**: Counters and histograms are recorded with attributes
5. **Error Recording**: Errors are automatically captured in spans

## Configuration

The example uses:

- gRPC protocol (default)
- Localhost endpoint
- Info log level
- Insecure connection (for local development)

For production, you'd add TLS:

```go
tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "collector.example.com:4317",
}.WithTrustedCA("/path/to/ca.crt"))
```

## Next Steps

Check out other examples:

- `examples/database/` - Database query tracing
- `examples/http-client/` - Outbound HTTP request tracing
- `examples/testing/` - How to test instrumented code
