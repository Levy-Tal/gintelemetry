# Worker & Background Jobs Example

This example demonstrates how to instrument background jobs, periodic tasks, and queue workers using gintelemetry with custom helper functions.

## What's Demonstrated

### 1. **Periodic Scraper** (`startPeriodicScraper`)

- Uses custom `WithBackgroundJob()` helper for tracing
- Runs every 15 seconds
- Shows manual logging, timing, error handling
- Records custom metrics about scrape results

### 2. **Queue Worker** (`startQueueWorker`)

- Processes messages from a queue
- Each message processed with the `WithBackgroundJob()` helper
- Shows message-specific attributes and metrics
- Demonstrates error handling and logging

### 3. **Health Checker** (`startHealthChecker`)

- Manual span creation with child spans
- Creates child spans for each dependency check
- Shows span status setting (OK/Error)
- Records health metrics for monitoring

## Key Pattern: Custom WithBackgroundJob Helper

This example shows how to build a custom helper function for background jobs:

```go
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
    
    // Record metrics and handle errors
    histogram := tel.Metric().Histogram("job.duration")
    histogram.Record(ctx, time.Since(start).Milliseconds(),
        metric.WithAttributes(
            tel.Attr().String("job", name),
            tel.Attr().Bool("success", err == nil),
        ),
    )
    
    if err != nil {
        tel.Trace().RecordError(ctx, err)
        tel.Log().Error(ctx, "job failed", "job", name, "error", err.Error())
    } else {
        tel.Log().Info(ctx, "job completed", "job", name)
    }
    
    return err
}
```

Usage:

```go
err := WithBackgroundJob(tel, context.Background(), "scraper.metrics", func(ctx context.Context) error {
    // Your job logic here
    return doWork(ctx)
})
```

## Running the Example

### Quick Start with Docker Compose (Recommended)

The easiest way to get started is using Docker Compose, which sets up Jaeger, Prometheus, and the OpenTelemetry Collector.

#### 1. Start the observability stack

```bash
cd examples/worker
docker compose up -d
```

This starts:

- **OpenTelemetry Collector** (ports 4317/4318) - Receives telemetry from application
- **Jaeger** (port 16686) - Trace visualization UI  
- **Prometheus** (port 9090) - Metrics visualization

The telemetry flow is: **Application → OTEL Collector → Jaeger/Prometheus**

#### 2. Run the example

```bash
go run main.go
```

The application will connect to `localhost:4317` and start sending telemetry.

#### 3. View telemetry

- **Traces**: <http://localhost:16686> (Jaeger UI)
- **Metrics**: <http://localhost:9090> (Prometheus UI)
- **Health**: <http://localhost:8080/health>

#### 4. Stop the stack

```bash
docker compose down
```

### Alternative: Manual Jaeger Setup

If you prefer not to use Docker Compose:

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/all-in-one:latest
```

Then run:

```bash
cd examples/worker
go run main.go
```

## Viewing Telemetry

### Traces in Jaeger

1. Open <http://localhost:16686>
2. Select **"worker-example"** from the service dropdown
3. Click **"Find Traces"**

You'll see traces for:

- **scraper.metrics** - Periodic scraper runs (every 15 seconds)
- **worker.process_message** - Queue message processing
- **healthcheck.dependencies** - Health checks (every 30 seconds)

Click on any trace to see:

- Span hierarchy (parent/child relationships)
- Attributes (job names, message IDs, etc.)
- Timing information
- Error details (when jobs fail)
- Logs correlated with spans

### Metrics in Prometheus

1. Open <http://localhost:9090>
2. Try these queries:

```promql
# Job durations
job_duration_bucket

# Job completions
job_completions_total

# Job failures
job_failures_total

# Messages processed
messages_processed_total

# Scraper metrics count
scraper_metrics_count
```

### Logs

Logs are output to both:

1. **Stdout** - Your terminal (JSON format)
2. **OTLP Collector** - View with `docker compose logs -f otel-collector`

All logs include trace/span IDs for correlation.

## Testing the Workers

### Enqueue Messages

```bash
# Enqueue a message
curl -X POST http://localhost:8080/enqueue \
  -H "Content-Type: application/json" \
  -d '{"order_id": "order-123", "action": "process"}'
```

Watch the logs and Jaeger to see the message being processed by the queue worker.

### Health Check

```bash
curl http://localhost:8080/health
```

## What to Notice

### 1. Background Job Traces

Each background job creates a **root span** (not connected to HTTP requests):

- Periodic scraper creates independent traces every 15 seconds
- Queue messages create independent traces per message
- Health checks create independent traces every 30 seconds

### 2. Custom Helper Pattern

The `WithBackgroundJob()` helper shows how to:

- Create consistent tracing patterns
- Automatically record metrics
- Handle errors uniformly
- Add structured logging

This pattern can be adapted for your specific needs.

### 3. Child Spans

The health checker demonstrates creating child spans:

```go
ctx, stop := tel.Trace().StartSpan(ctx, "healthcheck.dependencies")
defer stop()

for _, dep := range dependencies {
    depCtx, depStop := tel.Trace().StartSpan(ctx, "healthcheck."+dep)
    // Check dependency...
    depStop()
}
```

### 4. Metrics with Attributes

All metrics include relevant attributes for filtering:

```go
histogram := tel.Metric().Histogram("job.duration")
histogram.Record(ctx, duration.Milliseconds(),
    metric.WithAttributes(
        tel.Attr().String("job", name),
        tel.Attr().Bool("success", err == nil),
    ),
)
```

### 5. Error Handling

Errors are automatically:

- Recorded in spans
- Logged with context
- Counted in metrics

## Architecture

```
┌─────────────────────────────────────────────────┐
│              Worker Application                  │
│                                                  │
│  ┌──────────────┐  ┌──────────────┐            │
│  │   Periodic   │  │    Queue     │            │
│  │   Scraper    │  │   Worker     │            │
│  │  (15s loop)  │  │  (on demand) │            │
│  └──────────────┘  └──────────────┘            │
│                                                  │
│  ┌──────────────┐  ┌──────────────┐            │
│  │    Health    │  │  HTTP API    │            │
│  │   Checker    │  │  (enqueue)   │            │
│  │  (30s loop)  │  │              │            │
│  └──────────────┘  └──────────────┘            │
│                                                  │
│         ↓ All send telemetry via OTLP           │
└─────────────────────────────────────────────────┘
                      ↓
        ┌─────────────────────────┐
        │  OpenTelemetry Collector │
        └─────────────────────────┘
                ↓           ↓
        ┌──────────┐  ┌──────────┐
        │  Jaeger  │  │Prometheus│
        │ (traces) │  │(metrics) │
        └──────────┘  └──────────┘
```

## Configuration

The example uses:

- **gRPC protocol** (default)
- **Localhost endpoint** (`localhost:4317`)
- **Info log level**
- **Insecure connection** (for local development)

For production:

```go
tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
    ServiceName: "worker-service",
    Endpoint:    "collector.example.com:4317",
    Insecure:    false, // Use TLS in production
    LogLevel:    gintelemetry.LevelInfo,
    GlobalAttributes: map[string]string{
        "environment": "production",
        "region":      "us-east-1",
    },
})
```

## Building Your Own Helpers

This example demonstrates building custom helpers on top of gintelemetry's simple API. You can create helpers for:

- **WithSpan** - Execute function within a span
- **MeasureDuration** - Measure and record function duration
- **WithRetry** - Retry logic with tracing
- **WithCircuitBreaker** - Circuit breaker pattern with metrics
- **WithRateLimit** - Rate limiting with telemetry

The key is to use the core primitives:

- `tel.Trace().StartSpan()` - Create spans
- `tel.Log().Info/Error()` - Log events
- `tel.Metric().Counter/Histogram()` - Record metrics
- `tel.Trace().RecordError()` - Record errors

## Next Steps

- Check `examples/basic/` for simpler HTTP examples
- Check `examples/database/` for database instrumentation
- Check `examples/testing/` for testing patterns
- Read the main README for more custom helper examples
