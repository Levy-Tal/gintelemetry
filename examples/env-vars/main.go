package main

import (
	"context"
	"os"

	"github.com/Levy-Tal/gintelemetry"
	"github.com/gin-gonic/gin"
)

func main() {
	ctx := context.Background()

	// Set environment variables (normally these would be set externally)
	// This demonstrates how to use env vars without hardcoding in Config
	os.Setenv("OTEL_SERVICE_NAME", "env-vars-example")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	os.Setenv("OTEL_RESOURCE_ATTRIBUTES", "team=platform,environment=production,region=us-east-1,version=1.0.0")

	// Initialize with minimal config - everything comes from env vars!
	tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
		// ServiceName comes from OTEL_SERVICE_NAME
		// Endpoint comes from OTEL_EXPORTER_OTLP_ENDPOINT
		// Global attributes come from OTEL_RESOURCE_ATTRIBUTES
		Insecure: true, // Use insecure connection for local development
		LogLevel: gintelemetry.LevelInfo,
	})
	if err != nil {
		panic(err)
	}
	defer tel.Shutdown(ctx)

	tel.Log().Info(ctx, "service started with env-configured attributes",
		"note", "team, environment, region, and version are from env vars",
	)

	router.GET("/hello", func(c *gin.Context) {
		ctx := c.Request.Context()

		// All telemetry will include the global attributes from env vars
		tel.Log().Info(ctx, "handling request")

		tel.Metric().IncrementCounter(ctx, "requests.total",
			tel.Attr().String("endpoint", "hello"),
		)

		c.JSON(200, gin.H{
			"message": "Hello! Check Jaeger to see global attributes from env vars",
			"note":    "Look for team=platform, environment=production, region=us-east-1",
		})
	})

	router.GET("/config-override", func(c *gin.Context) {
		ctx := c.Request.Context()

		tel.Log().Info(ctx, "demonstrating config override")

		c.JSON(200, gin.H{
			"message": "Config attributes override env vars for the same key",
		})
	})

	tel.Log().Info(ctx, "server starting",
		"port", 8080,
		"tip", "Try: curl http://localhost:8080/hello",
	)

	router.Run(":8080")
}
