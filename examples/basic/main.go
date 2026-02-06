package main

import (
	"context"
	"time"

	"github.com/Levy-Tal/gintelemetry"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	// Initialize telemetry with basic configuration
	tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
		ServiceName: "basic-example",
		Endpoint:    "localhost:4317", // gRPC endpoint
		Insecure:    true,             // Use insecure connection for local development
		LogLevel:    gintelemetry.LevelInfo,
	})
	if err != nil {
		panic(err)
	}
	defer tel.Shutdown(ctx)

	// Simple hello endpoint
	router.GET("/hello", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Logging with automatic trace correlation
		tel.Log().Info(ctx, "handling hello request")

		// Increment counter with attributes - single import!
		tel.Metric().AddCounter(ctx, "requests.total", 1,
			tel.Attr().String("endpoint", "hello"),
			tel.Attr().String("method", "GET"),
		)

		c.JSON(200, gin.H{"message": "Hello, World!"})
	})

	// Example with manual span and error handling
	router.GET("/process/:id", func(c *gin.Context) {
		ctx := c.Request.Context()
		id := c.Param("id")

		// Create a span for this operation with attributes
		ctx, stop := tel.Trace().StartSpan(ctx, "process.item",
			tel.Attr().String("item.id", id),
		)
		defer stop()

		tel.Log().Info(ctx, "processing item", "item_id", id)

		// Simulate some work
		err := simulateWork(ctx, tel, id)
		if err != nil {
			tel.Trace().RecordError(ctx, err)
			tel.Log().Error(ctx, "processing failed", "error", err.Error())
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		tel.Metric().AddCounter(ctx, "items.processed", 1,
			tel.Attr().String("status", "success"),
		)

		c.JSON(200, gin.H{"status": "processed", "id": id})
	})

	// Example using custom helper
	router.GET("/measure", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Custom helper that measures duration
		err := measureDuration(tel, ctx, "operation.duration", func() error {
			time.Sleep(100 * time.Millisecond) // Simulate work
			return nil
		})

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "completed"})
	})

	tel.Log().Info(ctx, "server starting", "port", 8080)
	router.Run(":8080")
}

func simulateWork(ctx context.Context, tel *gintelemetry.Telemetry, id string) error {
	// Create a span for database query
	ctx, stop := tel.Trace().StartSpan(ctx, "database.query")
	defer stop()

	tel.Log().Debug(ctx, "querying database", "item_id", id)

	// Simulate database query
	start := time.Now()
	time.Sleep(50 * time.Millisecond)

	tel.Metric().RecordHistogram(ctx, "db.query.duration", time.Since(start).Milliseconds())

	return nil
}

// measureDuration is a custom helper that measures and records the duration of a function
func measureDuration(tel *gintelemetry.Telemetry, ctx context.Context, metricName string, fn func() error) error {
	start := time.Now()
	err := fn()

	tel.Metric().RecordHistogram(ctx, metricName, time.Since(start).Milliseconds())

	if err != nil {
		tel.Trace().RecordError(ctx, err)
	}

	return err
}
