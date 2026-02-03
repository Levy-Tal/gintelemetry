# Environment Variables Example

Demonstrates how to configure gintelemetry entirely through environment variables, including global attributes.

## What This Example Shows

- Using `OTEL_SERVICE_NAME` for service name
- Using `OTEL_EXPORTER_OTLP_ENDPOINT` for collector endpoint
- Using `OTEL_RESOURCE_ATTRIBUTES` for global attributes
- Zero hardcoded configuration in code
- Change team, environment, region without recompiling

## Why This Matters

With environment variables, you can:

- **Deploy the same binary** to different environments (dev/staging/prod)
- **Change team/region** without rebuilding
- **Use in containers** (Docker, Kubernetes) with different configs
- **CI/CD friendly** - configure via pipeline variables
- **Follow 12-factor app** principles

## Running the Example

### Start Jaeger

```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -e COLLECTOR_OTLP_ENABLED=true \
  jaegertracing/all-in-one:latest
```

### Run with Environment Variables

```bash
cd examples/env-vars

# Development environment
export OTEL_SERVICE_NAME=my-api
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
export OTEL_RESOURCE_ATTRIBUTES="team=backend,environment=dev,region=local,version=dev"
go run main.go
```

### Test It

```bash
curl http://localhost:8080/hello
```

### View in Jaeger

1. Open <http://localhost:16686>
2. Select your service from the dropdown
3. Click "Find Traces"
4. Click on a trace
5. Look at the **Process** section - you'll see:
   - `team=backend`
   - `environment=dev`
   - `region=local`
   - `version=dev`

## Different Environments

### Development

```bash
export OTEL_SERVICE_NAME=my-api-dev
export OTEL_RESOURCE_ATTRIBUTES="team=backend,environment=dev,region=local"
go run main.go
```

### Staging

```bash
export OTEL_SERVICE_NAME=my-api-staging
export OTEL_RESOURCE_ATTRIBUTES="team=backend,environment=staging,region=us-east-1"
go run main.go
```

### Production

```bash
export OTEL_SERVICE_NAME=my-api
export OTEL_RESOURCE_ATTRIBUTES="team=backend,environment=production,region=us-east-1,version=1.2.3"
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
  -e OTEL_RESOURCE_ATTRIBUTES="team=platform,environment=production,region=us-west-2" \
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
        - name: OTEL_RESOURCE_ATTRIBUTES
          value: "team=platform,environment=production,region=us-east-1,cluster=prod-1"
```

## OTEL_RESOURCE_ATTRIBUTES Format

**Format:** Comma-separated `key=value` pairs

```bash
export OTEL_RESOURCE_ATTRIBUTES="key1=value1,key2=value2,key3=value3"
```

**Example with common attributes:**

```bash
export OTEL_RESOURCE_ATTRIBUTES="team=platform,environment=production,region=us-east-1,version=1.2.3,deployment=blue"
```

**Special characters:** If you need commas or equals signs in values, escape them with backslash:

```bash
export OTEL_RESOURCE_ATTRIBUTES="description=Hello\, World,equation=2+2\=4"
```

## Precedence Rules

1. **Config attributes override env vars** for the same key
2. **Env vars fill in missing** config attributes

**Example:**

```go
// Config has team=frontend
config := gintelemetry.Config{
    GlobalAttributes: map[string]string{
        "team": "frontend",  // This wins
    },
}

// But env var has team=backend
export OTEL_RESOURCE_ATTRIBUTES="team=backend,environment=production"

// Result: team=frontend (from config), environment=production (from env)
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
  OTEL_RESOURCE_ATTRIBUTES: "environment=${{ github.event.inputs.environment }},version=${{ github.sha }}"
```

### 3. Use ConfigMaps in Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-config
data:
  OTEL_RESOURCE_ATTRIBUTES: "team=platform,environment=production,region=us-east-1"
```

### 4. Document Your Attributes

Create a standard attributes document for your organization so all services use consistent naming.

## Troubleshooting

### Attributes Not Showing Up

Check if they're being parsed correctly:

```go
tel.Log().Info(ctx, "startup", "attributes", "should be visible in trace")
```

Then look in Jaeger under the Process section.

### Environment Not Loading

Make sure you're setting the env var before starting your app:

```bash
# ❌ Wrong - env var set after process starts
go run main.go &
export OTEL_RESOURCE_ATTRIBUTES="team=platform"

# ✅ Correct - env var set before process starts
export OTEL_RESOURCE_ATTRIBUTES="team=platform"
go run main.go
```

## Next Steps

- Check `examples/basic/` for basic usage patterns
- Check `examples/database/` for database instrumentation
- Check `examples/testing/` for testing patterns
