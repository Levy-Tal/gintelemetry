# Database Example

Demonstrates how to instrument database operations with tracing and metrics using custom helper functions.

## What This Example Shows

- Database query tracing with spans
- Automatic duration measurement for queries
- Query-level attributes (operation, table, rows)
- Error recording for failed queries
- Query metrics (count, duration)
- Context propagation through database calls
- Custom `measureDuration` helper for timing
- Custom `WithSpan` generic helper for automatic span management

## Prerequisites

This example uses **Docker Compose** to run the observability stack:

```bash
cd examples/database
docker compose up -d
```

This starts:

- **OpenTelemetry Collector** - Receives all telemetry (traces, metrics, logs)
- **Jaeger UI** - Visualizes traces at <http://localhost:16686>

To stop:

```bash
docker compose down
```

## Running the Example

```bash
cd examples/database
go mod download
go run main.go
```

The example uses SQLite in-memory database with sample data pre-loaded.

## Try It Out

### List All Users

```bash
curl http://localhost:8080/users
```

Response:

```json
[
  {
    "id": 1,
    "name": "Alice Smith",
    "email": "alice@example.com",
    "created_at": "2024-01-01 12:00:00"
  }
]
```

### Get User by ID

```bash
curl http://localhost:8080/users/1
```

### Create New User

```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com"}'
```

## View the Results

### Traces in Jaeger UI

1. Open <http://localhost:16686>
2. Select **"database-example"** service
3. Click **"Find Traces"**
4. Explore traces to see:
   - `db.init` span (database initialization)
   - `db.seed` span (sample data insertion)
   - `db.query` spans with attributes:
     - `db.operation`: SELECT/INSERT
     - `db.table`: users
     - `db.rows_returned`: count of rows
   - Nested span relationships showing query flow

### Metrics and Logs

View metrics and logs in the OpenTelemetry Collector output:

```bash
docker compose logs -f otel-collector
```

You'll see:

- `db.insert.duration` and `db.query.duration` histogram metrics
- Application logs with trace correlation
- Database operation counts and timings

## Key Patterns

### Custom Helper: WithSpan (Generic)

This example demonstrates a generic helper that can return values:

```go
func WithSpan[T any](tel *gintelemetry.Telemetry, ctx context.Context, spanName string, fn func(context.Context) (T, error)) (T, error) {
    newCtx, stop := tel.Trace().StartSpan(ctx, spanName)
    defer stop()
    
    result, err := fn(newCtx)
    if err != nil {
        tel.Trace().RecordError(newCtx, err)
    }
    
    return result, err
}
```

Usage:

```go
db, err := WithSpan(tel, ctx, "db.init", func(ctx context.Context) (*sql.DB, error) {
    return sql.Open("sqlite3", ":memory:")
})
```

### Custom Helper: measureDuration

```go
func measureDuration(tel *gintelemetry.Telemetry, ctx context.Context, metricName string, fn func() error) error {
    start := time.Now()
    err := fn()
    
    histogram := tel.Metric().Histogram(metricName)
    histogram.Record(ctx, time.Since(start).Milliseconds())
    
    if err != nil {
        tel.Trace().RecordError(ctx, err)
    }
    
    return err
}
```

### Database Query Tracing

```go
func getUsers(ctx context.Context, tel *gintelemetry.Telemetry, db *sql.DB) ([]User, error) {
    // Create span with query details
    ctx, stop := tel.Trace().StartSpan(ctx, "db.query",
        trace.WithAttributes(
            tel.Attr().String("db.operation", "SELECT"),
            tel.Attr().String("db.table", "users"),
        ),
    )
    defer stop()

    // Measure query duration
    err := measureDuration(tel, ctx, "db.query.duration", func() error {
        // Execute query...
    })

    // Record results as attributes
    tel.Trace().SetAttributes(ctx,
        tel.Attr().Int("db.rows_returned", len(users)),
    )

    return users, nil
}
```

### Metrics for Query Monitoring

```go
counter := tel.Metric().Counter("db.queries.total")
counter.Add(ctx, 1, metric.WithAttributes(
    tel.Attr().String("operation", "SELECT"),
    tel.Attr().String("table", "users"),
    tel.Attr().String("status", "success"),
))
```

This allows you to:

- Count queries by operation type
- Track success/failure rates
- Monitor which tables are accessed most
- Identify slow queries

### Error Handling

```go
if err != nil {
    tel.Trace().RecordError(ctx, err)
    counter := tel.Metric().Counter("db.queries.total")
    counter.Add(ctx, 1, metric.WithAttributes(
        tel.Attr().String("status", "error"),
    ))
    return nil, err
}
```

## What to Look For in Traces

1. **Query Timing**: See how long each query takes
2. **Query Attributes**: Operation type, table name, row counts
3. **Nested Spans**: Queries within transactions within requests
4. **Error Details**: Failed queries show error messages and stack traces
5. **Correlation**: Logs appear alongside spans in the same trace

## Production Tips

### Use Connection Pooling

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Add Connection Pool Metrics

```go
stats := db.Stats()
gauge := tel.Metric().Gauge("db.connections.open")
gauge.Record(ctx, int64(stats.OpenConnections))
```

### Sanitize Query Parameters

Don't include sensitive data in span attributes:

```go
// ❌ BAD - exposes user data
tel.Attr().String("user.email", email)

// ✅ GOOD - use IDs only
tel.Attr().String("user.id", userID)
```

### Instrument at the Right Level

- ✅ DO: Instrument at the query/transaction level
- ❌ DON'T: Instrument individual row scans (too verbose)

## Next Steps

- Check `examples/basic/` for simpler examples
- Check `examples/worker/` for background job patterns
- Check `examples/testing/` for testing database code
