# Full Observability Stack Example

This example demonstrates a complete observability stack with:

- **Grafana Mimir** - Metrics storage and querying
- **Grafana Tempo** - Distributed tracing
- **Grafana Loki** - Log aggregation
- **OpenTelemetry Collector** - Telemetry data routing
- **Grafana** - Unified visualization dashboard

All components are pre-configured and ready to use!

## Architecture

```
┌─────────────────┐
│  Your App       │
│  (Port 8080)    │
└────────┬────────┘
         │ OTLP (gRPC)
         ▼
┌─────────────────┐
│ OTEL Collector  │
│  (Port 4317)    │
└────┬────┬───┬───┘
     │    │   │
     │    │   └──────────┐
     │    │              │
     ▼    ▼              ▼
┌─────┐ ┌─────┐    ┌─────┐
│Tempo│ │Loki │    │Mimir│
│3200 │ │3100 │    │9009 │
└──┬──┘ └──┬──┘    └──┬──┘
   │       │          │
   └───────┴──────────┘
           │
           ▼
      ┌─────────┐
      │ Grafana │
      │  :3000  │
      └─────────┘
```

## Quick Start

### 1. Start the Stack

```bash
cd examples/observability-stack
docker compose up -d
```

This will start all services:

- **Grafana**: <http://localhost:3000> (no login required)
- **Application**: <http://localhost:8080>
- **OTEL Collector**: localhost:4317 (gRPC)
- **Tempo**: <http://localhost:3200>
- **Loki**: <http://localhost:3100>
- **Mimir**: <http://localhost:9009>

### 2. Generate Some Traffic

```bash
# Health check
curl http://localhost:8080/health

# Simple request
curl http://localhost:8080/hello

# Complex request with traces
curl http://localhost:8080/process/12345

# Slow operation
curl http://localhost:8080/slow

# Generate an error
curl http://localhost:8080/error
```

Or use the load generator script:

```bash
# Generate continuous traffic
while true; do
  curl -s http://localhost:8080/process/$RANDOM > /dev/null
  curl -s http://localhost:8080/hello > /dev/null
  sleep 1
done
```

### 3. Explore in Grafana

Open <http://localhost:3000> in your browser (no login needed).

#### View Traces (Tempo)

1. Click **Explore** (compass icon) in the left sidebar
2. Select **Tempo** as the data source
3. Click **Search** tab
4. Select service: `observability-stack-example`
5. Click **Run query**
6. Click on any trace to see the full trace details with spans

**What to look for:**

- Request flow through your application
- Database queries, API calls, cache operations
- Timing information for each operation
- Error traces (try the `/error` endpoint)

#### View Logs (Loki)

1. Click **Explore**
2. Select **Loki** as the data source
3. Use LogQL queries:

```logql
# All logs from the app
{service_name="observability-stack-example"}

# Only error logs
{service_name="observability-stack-example"} |= "error"

# Logs for a specific trace (click "Logs for this span" in Tempo)
{service_name="observability-stack-example"} | trace_id="abc123..."
```

**What to look for:**

- Structured logs with context
- Automatic correlation with traces (click trace_id links)
- Log levels (info, error, debug)

#### View Metrics (Mimir)

1. Click **Explore**
2. Select **Mimir** as the data source
3. Use PromQL queries:

```promql
# Request rate
rate(requests_total[5m])

# Request rate by endpoint
sum by (endpoint) (rate(requests_total[5m]))

# Database query duration (p95)
histogram_quantile(0.95, rate(db_query_duration_bucket[5m]))

# Error rate
rate(errors_total[5m])

# Cache hit rate
sum(rate(cache_operations_total{hit="true"}[5m])) / 
sum(rate(cache_operations_total[5m]))

# Active connections
system_connections_active
```

**What to look for:**

- Request rates and throughput
- Latency percentiles (p50, p95, p99)
- Error rates
- Custom business metrics

#### Create a Dashboard

1. Click **Dashboards** → **New** → **New Dashboard**
2. Click **Add visualization**
3. Select **Mimir** as data source
4. Add panels with queries like:

**Request Rate Panel:**

```promql
sum(rate(requests_total[5m]))
```

**Error Rate Panel:**

```promql
sum(rate(errors_total[5m]))
```

**Latency Heatmap:**

```promql
rate(operation_duration_bucket[5m])
```

## Understanding the Data Flow

### Traces (Tempo)

The application creates spans for:

- HTTP requests (automatic via Gin middleware)
- Database queries
- External API calls
- Cache operations
- Background jobs

Each span includes:

- Operation name
- Start time and duration
- Attributes (key-value pairs)
- Parent-child relationships
- Error information

### Logs (Loki)

Logs are sent with:

- Structured fields (key-value pairs)
- Log levels (debug, info, warn, error)
- Automatic trace context (trace_id, span_id)
- Service metadata

### Metrics (Mimir)

The application records:

- **Counters**: `requests_total`, `errors_total`, `db_queries_total`
- **Histograms**: `db_query_duration`, `operation_duration`, `http_client_duration`
- **Gauges**: `users_count`, `system_cpu_usage`, `system_connections_active`

## Exploring Correlations

One of the most powerful features is the correlation between signals:

### From Traces to Logs

1. Open a trace in Tempo
2. Click on any span
3. Click **Logs for this span**
4. See all logs that occurred during that span

### From Logs to Traces

1. View logs in Loki
2. See `trace_id` field in log entries
3. Click the trace_id link
4. Jump directly to the trace in Tempo

### From Metrics to Traces

1. View a metric spike in Mimir
2. Note the time range
3. Switch to Tempo
4. Search for traces in that time range
5. Find the slow/error traces causing the spike

## Example Queries

### Find Slow Requests

In Tempo, use TraceQL:

```
{ duration > 500ms }
```

### Find Errors

```
{ status = error }
```

### Find Database Operations

```
{ name =~ "database.*" }
```

### Logs for Failed Requests

In Loki:

```logql
{service_name="observability-stack-example"} |= "error" | json
```

### Top Endpoints by Request Count

In Mimir:

```promql
topk(5, sum by (endpoint) (rate(requests_total[5m])))
```

## Configuration Files

- **`docker compose.yml`** - Orchestrates all services
- **`otel-collector-config.yml`** - Routes telemetry to backends
- **`tempo-config.yml`** - Tempo configuration
- **`loki-config.yml`** - Loki configuration (uses defaults)
- **`mimir-config.yml`** - Mimir configuration
- **`grafana-datasources.yml`** - Pre-configured data sources with correlations

## Customization

### Add Custom Metrics

```go
tel.Metric().AddCounter(ctx, "my_custom_metric", 1,
    tel.Attr().String("custom_label", "value"),
)
```

### Add Custom Spans

```go
ctx, stop := tel.Trace().StartSpan(ctx, "my.operation",
    tel.Attr().String("operation.type", "custom"),
)
defer stop()
```

### Add Custom Logs

```go
tel.Log().Info(ctx, "custom event",
    "user_id", userID,
    "action", "purchase",
)
```

## Troubleshooting

### Check Service Health

```bash
# Check all containers are running
docker compose ps

# Check collector logs
docker compose logs otel-collector

# Check application logs
docker compose logs app

# Check Grafana logs
docker compose logs grafana
```

### Verify Data Flow

```bash
# Check collector is receiving data (should see log entries)
docker compose logs otel-collector | grep -i "traces"
docker compose logs otel-collector | grep -i "metrics"
docker compose logs otel-collector | grep -i "logs"
```

### Reset Everything

```bash
# Stop and remove all containers and volumes
docker compose down -v

# Start fresh
docker compose up -d
```

## Production Considerations

This example is configured for **local development** with:

- Anonymous Grafana access
- Insecure connections
- In-memory/local storage
- Single-instance deployments
- Short retention periods

For production, you should:

- Enable authentication and TLS
- Use persistent storage (S3, GCS, etc.)
- Scale components horizontally
- Configure proper retention policies
- Set up alerting rules
- Use proper resource limits
- Implement backup strategies

## Learn More

- [Grafana Documentation](https://grafana.com/docs/)
- [Tempo Documentation](https://grafana.com/docs/tempo/)
- [Loki Documentation](https://grafana.com/docs/loki/)
- [Mimir Documentation](https://grafana.com/docs/mimir/)
- [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/)
- [TraceQL Query Language](https://grafana.com/docs/tempo/latest/traceql/)
- [LogQL Query Language](https://grafana.com/docs/loki/latest/logql/)
- [PromQL Query Language](https://prometheus.io/docs/prometheus/latest/querying/basics/)

## Cleanup

```bash
# Stop all services
docker compose down

# Remove volumes (deletes all data)
docker compose down -v
```

---

**Enjoy exploring your observability stack!** 🎉

Try generating different types of traffic and see how the data flows through the system. The pre-configured correlations make it easy to jump between traces, logs, and metrics to understand what's happening in your application.
