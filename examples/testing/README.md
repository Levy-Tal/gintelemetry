# Testing Example

Demonstrates how to write tests for code instrumented with gintelemetry.

## What This Example Shows

- Using `NewTestConfig()` for testing
- Testing services with telemetry
- Testing HTTP handlers with telemetry
- Testing with custom contexts
- Concurrent request testing
- Benchmarking instrumented code
- Tests work without a collector running

## Running the Tests

```bash
cd examples/testing
go test -v
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Key Testing Patterns

### 1. Basic Service Testing

```go
func TestOrderService_ProcessOrder(t *testing.T) {
    ctx := context.Background()

    // Create test telemetry - no collector needed!
    tel, _, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-service"))
    if err != nil {
        t.Fatalf("failed to start telemetry: %v", err)
    }
    defer tel.Shutdown(ctx)

    service := NewOrderService(tel)

    // Test your service - telemetry calls work but don't require collector
    err = service.ProcessOrder(ctx, "order-123")
    if err != nil {
        t.Errorf("ProcessOrder failed: %v", err)
    }
}
```

### 2. HTTP Handler Testing

```go
func TestHandlers(t *testing.T) {
    ctx := context.Background()

    // Start returns a router with telemetry middleware already configured
    tel, router, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-handlers"))
    if err != nil {
        t.Fatalf("failed to start telemetry: %v", err)
    }
    defer tel.Shutdown(ctx)

    // Setup your routes
    SetupRoutes(router, tel)

    // Test with httptest
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("unexpected status: got %v want %v", w.Code, http.StatusOK)
    }
}
```

### 3. Testing with Custom Context

```go
func TestWithCustomContext(t *testing.T) {
    ctx := context.Background()

    tel, _, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-context"))
    if err != nil {
        t.Fatalf("failed to start telemetry: %v", err)
    }
    defer tel.Shutdown(ctx)

    // Create a test span
    ctx, stop := tel.Trace().StartSpan(ctx, "test.span")
    defer stop()

    // Add test attributes
    tel.Trace().SetAttributes(ctx,
        tel.Attr().String("test.name", "my test"),
    )

    // Use the context in your tests
    service := NewOrderService(tel)
    err = service.ProcessOrder(ctx, "test-order")
    // ...
}
```

### 4. Testing Concurrent Operations

```go
func TestConcurrentRequests(t *testing.T) {
    tel, router, _ := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-concurrent"))
    defer tel.Shutdown(ctx)

    const numRequests = 10
    done := make(chan bool, numRequests)

    for i := 0; i < numRequests; i++ {
        go func() {
            req := httptest.NewRequest("GET", "/health", nil)
            w := httptest.NewRecorder()
            router.ServeHTTP(w, req)
            done <- true
        }()
    }

    for i := 0; i < numRequests; i++ {
        <-done
    }
}
```

### 5. Benchmarking

```go
func BenchmarkProcessOrder(b *testing.B) {
    ctx := context.Background()

    tel, _, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("bench-service"))
    if err != nil {
        b.Fatalf("failed to start telemetry: %v", err)
    }
    defer tel.Shutdown(ctx)

    service := NewOrderService(tel)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = service.ProcessOrder(ctx, "order-123")
    }
}
```

## What NewTestConfig Does

`NewTestConfig()` creates a configuration that:

- ✅ Points to localhost (works even without collector)
- ✅ Sets log level to Error (reduces test noise)
- ✅ Uses insecure localhost connection
- ✅ Safe to use even without a collector running

## Best Practices

### 1. Always Shutdown Telemetry

```go
tel, _, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test"))
if err != nil {
    t.Fatalf("failed to start: %v", err)
}
defer tel.Shutdown(ctx) // Important!
```

### 2. Use Table-Driven Tests

```go
tests := []struct {
    name    string
    orderID string
    wantErr bool
}{
    {"valid order", "order-123", false},
    {"empty order", "", true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        err := service.ProcessOrder(ctx, tt.orderID)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
        }
    })
}
```

### 3. Test Both Success and Error Cases

```go
// Test success
err := service.ProcessOrder(ctx, "valid-order")
if err != nil {
    t.Errorf("expected success, got error: %v", err)
}

// Test error
err = service.ProcessOrder(ctx, "")
if err == nil {
    t.Error("expected error for empty order ID")
}
```

### 4. Don't Test Telemetry Implementation

Focus on your business logic, not whether telemetry calls were made:

```go
// ✅ GOOD - Test business logic
if status != "completed" {
    t.Errorf("wrong status: got %v want completed", status)
}

// ❌ BAD - Don't test telemetry internals
// if spanWasCreated { ... } // Don't do this
```

### 5. Use Test-Specific Service Names

```go
// Makes it easier to identify test traces if you do use a real collector
gintelemetry.NewTestConfig("test-my-feature")
```

## Integration Testing

If you want to test against a real collector:

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Use real config instead of NewTestConfig
    tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
        ServiceName: "integration-test",
        Endpoint:    "localhost:4317",
        Insecure:    true,
    })
    if err != nil {
        t.Skipf("collector not available: %v", err)
    }
    defer tel.Shutdown(ctx)

    // Test with real telemetry...
}
```

Run integration tests:

```bash
# Skip integration tests
go test -short

# Run all tests including integration
go test
```

## Common Pitfalls

### ❌ Forgetting to Shutdown

```go
tel, _, _ := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test"))
// Missing: defer tel.Shutdown(ctx)
// Can cause resource leaks in test suites
```

### ❌ Reusing Telemetry Across Tests

```go
// DON'T - Global telemetry shared across tests
var globalTel *gintelemetry.Telemetry

func TestA(t *testing.T) { /* uses globalTel */ }
func TestB(t *testing.T) { /* uses globalTel */ }
```

Each test should have its own telemetry instance for isolation.

### ❌ Testing with context.Background() Only

```go
// Missing: Pass request context through handlers
ctx := context.Background()
service.ProcessOrder(ctx, "order-123")
```

In real code, use `c.Request.Context()` to get trace correlation.

## Running the Example

You can also run this as a real service:

```bash
cd examples/testing
go run .
```

Then test the endpoints:

```bash
curl -X POST http://localhost:8080/orders/123/process
curl http://localhost:8080/orders/123/status
```

## Next Steps

- Check `examples/basic/` for basic usage patterns
- Check `examples/database/` for database testing patterns
- Review the main README for testing documentation
