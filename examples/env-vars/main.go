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

	// Initialize with minimal config - everything comes from env vars!
	tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
		// ServiceName comes from OTEL_SERVICE_NAME
		// Endpoint comes from OTEL_EXPORTER_OTLP_ENDPOINT
		Insecure: true, // Use insecure connection for local development
		LogLevel: gintelemetry.LevelInfo,
		GlobalAttributes: map[string]string{
			"team":        "platform",
			"environment": "production",
			"region":      "us-east-1",
			"version":     "1.0.0",
		},
	})
	if err != nil {
		panic(err)
	}
	defer tel.Shutdown(ctx)

	tel.Log().Info(ctx, "service started with configured attributes",
		"note", "team, environment, region, and version are global attributes",
	)

	router.GET("/hello", func(c *gin.Context) {
		ctx := c.Request.Context()

		// All telemetry will include the global attributes
		tel.Log().Info(ctx, "handling request")

		tel.Metric().AddCounter(ctx, "requests.total", 1,
			tel.Attr().String("endpoint", "hello"),
		)

		c.JSON(200, gin.H{
			"message": "Hello! Check Jaeger to see global attributes",
			"note":    "Look for team=platform, environment=production, region=us-east-1",
		})
	})

	router.GET("/config-override", func(c *gin.Context) {
		ctx := c.Request.Context()

		tel.Log().Info(ctx, "demonstrating config override")

		c.JSON(200, gin.H{
			"message": "Global attributes are set in Config.GlobalAttributes",
		})
	})

	tel.Log().Info(ctx, "server starting",
		"port", 8080,
		"tip", "Try: curl http://localhost:8080/hello",
	)

	router.Run(":8080")
}
