# Environment Variables Example

Demonstrates how to configure gintelemetry through environment variables and Config struct.

## What This Example Shows

- Using `OTEL_SERVICE_NAME` for service name
- Using `OTEL_EXPORTER_OTLP_ENDPOINT` for collector endpoint
- Setting global attributes in Config
- Zero hardcoded endpoint configuration
- Change service name without recompiling

## Why This Matters

With environment variables, you can:

- **Deploy the same binary** to different environments (dev/staging/prod)
- **Change endpoints** without rebuilding
- **Use in containers** (Docker, Kubernetes) with different configs
- **CI/CD friendly** - configure via pipeline variables
- **Follow 12-factor app** principles

## Running the Example

### Start the Observability Stack

```bash
cd examples/env-vars
docker compose up -d
```

This starts:

- **OpenTelemetry Collector** on ports 4317 (gRPC) and 4318 (HTTP)
- **Jaeger UI** at <http://localhost:16686>

### Run with Environment Variables

```bash
cd examples/env-vars

# Development environment
export OTEL_SERVICE_NAME=my-api
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
go run main.go
```

### Test It

```bash
curl http://localhost:8080/hello
```

### View Results

**Jaeger UI** (<http://localhost:16686>):

1. Select your service from the dropdown
2. Click "Find Traces"
3. Click on a trace
4. Look at the **Process** section - you'll see your global attributes:
   - `team=platform`
   - `environment=production`
   - `region=us-east-1`
   - `version=1.0.0`

**Collector Logs** (metrics and logs):

```bash
docker compose logs -f otel-collector
```

## Different Environments

### Development

```bash
export OTEL_SERVICE_NAME=my-api-dev
go run main.go
```

### Staging

```bash
export OTEL_SERVICE_NAME=my-api-staging
export OTEL_EXPORTER_OTLP_ENDPOINT=staging-collector:4317
go run main.go
```

### Production

```bash
export OTEL_SERVICE_NAME=my-api
export OTEL_EXPORTER_OTLP_ENDPOINT=prod-collector:4317
go run main.go
```

**Same binary, different configuration!**

## Docker Example

### Dockerfile

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o app main.go

FROM gcr.io/distroless/base-debian12
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
```

### Running with Docker

```bash
# Build once
docker build -t my-api .

# Run in different environments
docker run -e OTEL_SERVICE_NAME=my-api \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317 \
  my-api
```

## Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-api
spec:
  template:
    spec:
      containers:
      - name: my-api
        image: my-api:latest
        env:
        - name: OTEL_SERVICE_NAME
          value: "my-api"
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "otel-collector:4317"
```

## Configuration Options

### Via Environment Variables

```bash
export OTEL_SERVICE_NAME=my-service
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

### Via Config Struct

```go
tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
    ServiceName: "my-service",
    Endpoint:    "localhost:4317",
    Insecure:    true,
    GlobalAttributes: map[string]string{
        "team":        "platform",
        "environment": "production",
        "region":      "us-east-1",
        "version":     "1.0.0",
    },
})
```

### Combining Both

Environment variables are used as fallbacks if Config fields are empty:

```go
// Empty config - reads from env vars
tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
    // ServiceName comes from OTEL_SERVICE_NAME
    // Endpoint comes from OTEL_EXPORTER_OTLP_ENDPOINT
    Insecure: true,
})
```

## Best Practices

### 1. Use Consistent Attribute Names

Standardize across your organization:

- `team` - Team name (backend, frontend, platform)
- `environment` - Environment (dev, staging, production)
- `region` - Region (us-east-1, eu-west-1, local)
- `version` - Application version (1.2.3, v2.0.0)
- `cluster` - Cluster identifier (prod-1, staging-a)

### 2. Set in CI/CD

```yaml
# GitHub Actions example
env:
  OTEL_SERVICE_NAME: ${{ github.event.repository.name }}
  OTEL_EXPORTER_OTLP_ENDPOINT: ${{ secrets.OTEL_ENDPOINT }}
```

### 3. Use ConfigMaps in Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-config
data:
  OTEL_SERVICE_NAME: "my-api"
  OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector:4317"
```

### 4. Document Your Attributes

Create a standard attributes document for your organization so all services use consistent naming.

## Troubleshooting

### Environment Not Loading

Make sure you're setting the env var before starting your app:

```bash
# ❌ Wrong - env var set after process starts
go run main.go &
export OTEL_SERVICE_NAME=my-service

# ✅ Correct - env var set before process starts
export OTEL_SERVICE_NAME=my-service
go run main.go
```

## Next Steps

- Check `examples/basic/` for basic usage patterns
- Check `examples/database/` for database instrumentation
- Check `examples/testing/` for testing patterns
