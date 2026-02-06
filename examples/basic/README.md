# Basic Example

A simple hello world example demonstrating core gintelemetry features.

## What This Example Shows

- Basic telemetry setup with `Start()`
- Automatic request tracing via middleware
- Logging with trace correlation
- Counter metrics with attributes
- Manual span creation
- Error recording in spans
- Custom helper functions (`measureDuration`)
- Unified attribute API

## Prerequisites

This example uses **Docker Compose** to run both the OpenTelemetry Collector and Jaeger:

- **OpenTelemetry Collector** - Receives all telemetry (traces, metrics, logs) from your app
- **Jaeger** - Stores and visualizes traces

### Start the Observability Stack

```bash
cd examples/basic
docker compose up -d
```

This starts:

- **OpenTelemetry Collector** on ports 4317 (gRPC) and 4318 (HTTP)
- **Jaeger UI** on port 16686

### Stop the Stack

```bash
docker compose down
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

- Custom `measureDuration` helper function
- Automatic duration recording

## View the Results

### Traces in Jaeger UI

1. Open <http://localhost:16686>
2. Select **"basic-example"** from the service dropdown
3. Click **"Find Traces"**
4. Click on any trace to see spans, logs, and attributes

You'll see:

- Request traces with all spans
- Span attributes (endpoint, method, item.id, etc.)
- Timing information
- Error details (if you trigger an error)

### Logs and Metrics

Logs and metrics are sent to the OpenTelemetry Collector and output to its console:

```bash
# View collector logs (includes metrics and logs)
docker compose logs -f otel-collector
```

You'll see your application logs and metrics in the collector output.

## What to Notice

1. **Trace Correlation**: Logs include trace/span IDs automatically
2. **Nested Spans**: The `process/:id` endpoint creates a parent span, then a child "database.query" span
3. **Attributes**: All spans include relevant attributes (ids, endpoints, etc.)
4. **Metrics**: Counters and histograms are recorded with attributes
5. **Error Recording**: Errors are automatically captured in spans
6. **Custom Helpers**: The example shows how to build your own helper functions on top of the simple API

## Configuration

The example uses:

- gRPC protocol (default)
- Localhost endpoint
- Info log level
- Insecure connection (for local development)

For production, you'd configure TLS through OpenTelemetry SDK environment variables or use secure endpoints.

## Next Steps

Check out other examples:

- `examples/database/` - Database query tracing with custom helpers
- `examples/worker/` - Background jobs and workers
- `examples/testing/` - How to test instrumented code
