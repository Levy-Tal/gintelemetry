package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Levy-Tal/gintelemetry"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	// Initialize telemetry - endpoint comes from environment variable
	tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
		ServiceName: "observability-stack-example",
		Endpoint:    "otel-collector:4317",
		Insecure:    true,
		LogLevel:    gintelemetry.LevelInfo,
		GlobalAttributes: map[string]string{
			"environment": "demo",
			"version":     "1.0.0",
		},
	})
	if err != nil {
		panic(err)
	}
	defer tel.Shutdown(ctx)

	tel.Log().Info(ctx, "🚀 Observability stack example started")

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Simple endpoint with metrics
	router.GET("/hello", func(c *gin.Context) {
		ctx := c.Request.Context()

		tel.Log().Info(ctx, "handling hello request")

		tel.Metric().AddCounter(ctx, "requests.total", 1,
			tel.Attr().String("endpoint", "/hello"),
			tel.Attr().String("method", "GET"),
		)

		c.JSON(200, gin.H{
			"message": "Hello from Observability Stack!",
			"tip":     "Check Grafana at http://localhost:3000",
		})
	})

	// Endpoint that simulates various operations
	router.GET("/process/:id", func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		// Create a span for this operation
		ctx, stop := tel.Trace().StartSpan(ctx, "process.request",
			tel.Attr().String("request.id", id),
		)
		defer stop()

		tel.Log().Info(ctx, "processing request", "request_id", id)

		// Simulate database query
		result, err := simulateDBQuery(ctx, tel, id)
		if err != nil {
			tel.Trace().RecordError(ctx, err)
			tel.Log().Error(ctx, "database query failed", "error", err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Simulate external API call
		apiData, err := simulateAPICall(ctx, tel)
		if err != nil {
			tel.Trace().RecordError(ctx, err)
			tel.Log().Warn(ctx, "API call failed, using fallback", "error", err.Error())
			apiData = "fallback-data"
		}

		// Simulate cache operation
		cacheHit := simulateCacheCheck(ctx, tel, id)

		tel.Metric().AddCounter(ctx, "requests.processed", 1,
			tel.Attr().String("status", "success"),
			tel.Attr().Bool("cache_hit", cacheHit),
		)

		tel.Log().Info(ctx, "request processed successfully",
			"request_id", id,
			"cache_hit", cacheHit,
		)

		c.JSON(200, gin.H{
			"id":        id,
			"result":    result,
			"api_data":  apiData,
			"cache_hit": cacheHit,
		})
	})

	// Endpoint that generates errors
	router.GET("/error", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, stop := tel.Trace().StartSpan(ctx, "error.endpoint")
		defer stop()

		tel.Log().Error(ctx, "intentional error for testing")

		tel.Metric().AddCounter(ctx, "errors.total", 1,
			tel.Attr().String("endpoint", "/error"),
		)

		err := fmt.Errorf("this is a test error")
		tel.Trace().RecordError(ctx, err)

		c.JSON(500, gin.H{"error": err.Error()})
	})

	// Endpoint with slow operations
	router.GET("/slow", func(c *gin.Context) {
		ctx := c.Request.Context()

		ctx, stop := tel.Trace().StartSpan(ctx, "slow.operation")
		defer stop()

		tel.Log().Info(ctx, "starting slow operation")

		// Simulate slow operation
		duration := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
		time.Sleep(duration)

		tel.Metric().RecordHistogram(ctx, "operation.duration", duration.Milliseconds(),
			tel.Attr().String("operation", "slow"),
		)

		tel.Log().Info(ctx, "slow operation completed",
			"duration_ms", duration.Milliseconds(),
		)

		c.JSON(200, gin.H{
			"duration_ms": duration.Milliseconds(),
			"message":     "Operation completed",
		})
	})

	// Start background job that generates telemetry
	go backgroundJob(tel)

	tel.Log().Info(ctx, "server starting",
		"port", 8080,
		"grafana", "http://localhost:3000",
	)

	router.Run(":8080")
}

func simulateDBQuery(ctx context.Context, tel *gintelemetry.Telemetry, id string) (string, error) {
	ctx, stop := tel.Trace().StartSpan(ctx, "database.query",
		tel.Attr().String("db.system", "postgresql"),
		tel.Attr().String("db.operation", "SELECT"),
		tel.Attr().String("db.table", "users"),
	)
	defer stop()

	start := time.Now()

	// Simulate query time
	queryDuration := time.Duration(10+rand.Intn(90)) * time.Millisecond
	time.Sleep(queryDuration)

	tel.Metric().RecordHistogram(ctx, "db.query.duration", queryDuration.Milliseconds(),
		tel.Attr().String("db.table", "users"),
		tel.Attr().String("db.operation", "SELECT"),
	)

	// Simulate occasional errors
	if rand.Float32() < 0.05 {
		err := fmt.Errorf("database connection timeout")
		tel.Trace().RecordError(ctx, err)
		tel.Metric().AddCounter(ctx, "db.errors.total", 1,
			tel.Attr().String("error_type", "timeout"),
		)
		return "", err
	}

	tel.Log().Debug(ctx, "database query completed",
		"duration_ms", time.Since(start).Milliseconds(),
		"rows_returned", 1,
	)

	tel.Metric().AddCounter(ctx, "db.queries.total", 1,
		tel.Attr().String("status", "success"),
	)

	return fmt.Sprintf("data-%s", id), nil
}

func simulateAPICall(ctx context.Context, tel *gintelemetry.Telemetry) (string, error) {
	ctx, stop := tel.Trace().StartSpanWithKind(ctx, "http.client.request",
		gintelemetry.SpanKindClient,
		tel.Attr().String("http.method", "GET"),
		tel.Attr().String("http.url", "https://api.example.com/data"),
	)
	defer stop()

	start := time.Now()

	// Simulate API call time
	apiDuration := time.Duration(50+rand.Intn(200)) * time.Millisecond
	time.Sleep(apiDuration)

	tel.Metric().RecordHistogram(ctx, "http.client.duration", apiDuration.Milliseconds(),
		tel.Attr().String("http.method", "GET"),
	)

	// Simulate occasional failures
	if rand.Float32() < 0.1 {
		err := fmt.Errorf("API request failed: 503 Service Unavailable")
		tel.Trace().RecordError(ctx, err)
		tel.Metric().AddCounter(ctx, "http.client.errors.total", 1,
			tel.Attr().String("error_type", "503"),
		)
		return "", err
	}

	tel.Log().Debug(ctx, "API call completed",
		"duration_ms", time.Since(start).Milliseconds(),
	)

	tel.Metric().AddCounter(ctx, "http.client.requests.total", 1,
		tel.Attr().String("status", "success"),
	)

	return "api-response-data", nil
}

func simulateCacheCheck(ctx context.Context, tel *gintelemetry.Telemetry, key string) bool {
	ctx, stop := tel.Trace().StartSpan(ctx, "cache.get",
		tel.Attr().String("cache.key", key),
	)
	defer stop()

	// Simulate cache lookup
	time.Sleep(time.Duration(1+rand.Intn(5)) * time.Millisecond)

	hit := rand.Float32() > 0.3 // 70% cache hit rate

	tel.Metric().AddCounter(ctx, "cache.operations.total", 1,
		tel.Attr().String("operation", "get"),
		tel.Attr().Bool("hit", hit),
	)

	if hit {
		tel.Log().Debug(ctx, "cache hit", "key", key)
		tel.Trace().AddEvent(ctx, "cache.hit",
			tel.Attr().String("cache.key", key),
		)
	} else {
		tel.Log().Debug(ctx, "cache miss", "key", key)
		tel.Trace().AddEvent(ctx, "cache.miss",
			tel.Attr().String("cache.key", key),
		)
	}

	return hit
}

func backgroundJob(tel *gintelemetry.Telemetry) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, stop := tel.Trace().StartSpanWithKind(context.Background(), "background.metrics_collector",
			gintelemetry.SpanKindInternal,
			tel.Attr().String("job.type", "metrics_collector"),
		)

		tel.Log().Info(ctx, "collecting system metrics")

		// Simulate collecting various metrics
		cpuUsage := 20 + rand.Float64()*60
		memoryUsage := 40 + rand.Float64()*40
		activeConnections := rand.Int63n(100)

		tel.Metric().RecordFloat64Gauge(ctx, "system.cpu.usage", cpuUsage,
			tel.Attr().String("unit", "percent"),
		)

		tel.Metric().RecordFloat64Gauge(ctx, "system.memory.usage", memoryUsage,
			tel.Attr().String("unit", "percent"),
		)

		tel.Metric().RecordGauge(ctx, "system.connections.active", activeConnections)

		tel.Log().Info(ctx, "metrics collected",
			"cpu_usage", fmt.Sprintf("%.2f%%", cpuUsage),
			"memory_usage", fmt.Sprintf("%.2f%%", memoryUsage),
			"active_connections", activeConnections,
		)

		stop()
	}
}
