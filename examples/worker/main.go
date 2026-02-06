package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Levy-Tal/gintelemetry"
	"github.com/gin-gonic/gin"
)

// Message represents a message in the queue
type Message struct {
	ID      string
	OrderID string
	Action  string
}

// SimpleQueue simulates a message queue
type SimpleQueue struct {
	messages chan Message
}

func NewQueue() *SimpleQueue {
	return &SimpleQueue{
		messages: make(chan Message, 100),
	}
}

func (q *SimpleQueue) Enqueue(msg Message) {
	q.messages <- msg
}

func (q *SimpleQueue) Messages() <-chan Message {
	return q.messages
}

func main() {
	ctx := context.Background()

	// Initialize telemetry
	tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
		ServiceName: "worker-example",
		Endpoint:    "localhost:4317",
		Insecure:    true,
		LogLevel:    gintelemetry.LevelInfo,
	})
	if err != nil {
		panic(err)
	}
	defer tel.Shutdown(ctx)

	// Create a simple queue
	queue := NewQueue()

	// Start background workers
	startPeriodicScraper(tel)
	startQueueWorker(tel, queue)
	startHealthChecker(tel)

	// API endpoint to enqueue messages
	router.POST("/enqueue", func(c *gin.Context) {
		var msg Message
		if err := c.BindJSON(&msg); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
		queue.Enqueue(msg)

		tel.Log().Info(c.Request.Context(), "message enqueued",
			"message_id", msg.ID,
			"order_id", msg.OrderID,
		)

		c.JSON(200, gin.H{"message_id": msg.ID, "status": "enqueued"})
	})

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	tel.Log().Info(ctx, "Worker example service starting on :8080")
	router.Run(":8080")
}

// WithBackgroundJob is a custom helper that wraps background jobs with tracing, logging, and metrics
func WithBackgroundJob(tel *gintelemetry.Telemetry, ctx context.Context, name string, fn func(context.Context) error) error {
	start := time.Now()

	// Create a new root span for the background job
	ctx, stop := tel.Trace().StartSpanWithKind(ctx, name,
		gintelemetry.SpanKindInternal,
		tel.Attr().String("job.type", "background"),
	)
	defer stop()

	tel.Log().Info(ctx, "job started", "job", name)

	err := fn(ctx)

	duration := time.Since(start)
	tel.Metric().RecordHistogram(ctx, "job.duration", duration.Milliseconds(),
		tel.Attr().String("job", name),
		tel.Attr().Bool("success", err == nil),
	)

	if err != nil {
		tel.Trace().RecordError(ctx, err)
		tel.Log().Error(ctx, "job failed", "job", name, "error", err.Error())

		tel.Metric().AddCounter(ctx, "job.failures", 1,
			tel.Attr().String("job", name),
		)
	} else {
		tel.Log().Info(ctx, "job completed", "job", name)

		tel.Metric().AddCounter(ctx, "job.completions", 1,
			tel.Attr().String("job", name),
		)
	}

	return err
}

// startPeriodicScraper demonstrates a periodic background job using the custom helper
func startPeriodicScraper(tel *gintelemetry.Telemetry) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Use custom WithBackgroundJob helper
			_ = WithBackgroundJob(tel, context.Background(), "scraper.metrics", func(ctx context.Context) error {
				// Add custom attributes
				tel.Trace().SetAttributes(ctx,
					tel.Attr().String("scrape.target", "prometheus:9090"),
					tel.Attr().String("scrape.type", "metrics"),
				)

				// Simulate work with random duration
				duration := time.Duration(100+rand.Intn(400)) * time.Millisecond
				time.Sleep(duration)

				// Simulate occasional failures
				if rand.Float32() < 0.1 {
					return fmt.Errorf("scrape timeout after %v", duration)
				}

				// Record metrics about the scrape
				metricsScraped := rand.Intn(100) + 50
				tel.Metric().RecordGauge(ctx, "scraper.metrics_count", int64(metricsScraped),
					tel.Attr().String("target", "prometheus"),
				)

				tel.Log().Info(ctx, "metrics scrape completed", "metrics_count", metricsScraped)
				return nil
			})
		}
	}()
}

// startQueueWorker demonstrates a queue worker pattern
func startQueueWorker(tel *gintelemetry.Telemetry, queue *SimpleQueue) {
	go func() {
		tel.Log().Info(context.Background(), "queue worker started")

		for msg := range queue.Messages() {
			// Process each message with the custom helper
			_ = WithBackgroundJob(tel, context.Background(), "worker.process_message", func(ctx context.Context) error {
				// Add message-specific attributes
				tel.Trace().SetAttributes(ctx,
					tel.Attr().String("message.id", msg.ID),
					tel.Attr().String("order.id", msg.OrderID),
					tel.Attr().String("message.action", msg.Action),
				)

				tel.Log().Info(ctx, "processing message",
					"message_id", msg.ID,
					"order_id", msg.OrderID,
					"action", msg.Action,
				)

				// Simulate processing work
				processingTime := time.Duration(50+rand.Intn(200)) * time.Millisecond
				time.Sleep(processingTime)

				// Simulate occasional failures
				if rand.Float32() < 0.15 {
					return fmt.Errorf("failed to process order %s", msg.OrderID)
				}

				// Record success metrics
				tel.Metric().AddCounter(ctx, "messages.processed", 1,
					tel.Attr().String("action", msg.Action),
				)

				tel.Log().Info(ctx, "message processed successfully",
					"message_id", msg.ID,
					"processing_time_ms", processingTime.Milliseconds(),
				)

				return nil
			})
		}
	}()
}

// startHealthChecker demonstrates a periodic health check pattern
func startHealthChecker(tel *gintelemetry.Telemetry) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Manual span creation for more control
			ctx, stop := tel.Trace().StartSpanWithKind(context.Background(), "healthcheck.dependencies",
				gintelemetry.SpanKindInternal,
				tel.Attr().String("job.type", "background"),
			)

			tel.Log().Info(ctx, "running health checks")

			// Check multiple dependencies
			dependencies := []string{"database", "redis", "external-api"}
			healthy := 0
			unhealthy := 0

			for _, dep := range dependencies {
				// Create child span for each dependency check
				depCtx, depStop := tel.Trace().StartSpan(ctx, "healthcheck."+dep)

				tel.Trace().SetAttributes(depCtx,
					tel.Attr().String("dependency", dep),
				)

				// Simulate health check
				checkDuration := time.Duration(10+rand.Intn(90)) * time.Millisecond
				time.Sleep(checkDuration)

				isHealthy := rand.Float32() > 0.2 // 80% healthy

				if isHealthy {
					healthy++
					tel.Trace().SetStatus(depCtx, gintelemetry.StatusOK, "healthy")
					tel.Log().Debug(depCtx, "dependency healthy", "dependency", dep)
				} else {
					unhealthy++
					tel.Trace().SetStatus(depCtx, gintelemetry.StatusError, "unhealthy")
					tel.Log().Warn(depCtx, "dependency unhealthy", "dependency", dep)
				}

				tel.Metric().RecordGauge(depCtx, "dependency.health",
					int64(map[bool]int{true: 1, false: 0}[isHealthy]),
					tel.Attr().String("dependency", dep),
				)

				depStop()
			}

			// Record overall health metrics
			tel.Metric().RecordGauge(ctx, "healthcheck.healthy_count", int64(healthy))
			tel.Metric().RecordGauge(ctx, "healthcheck.unhealthy_count", int64(unhealthy))

			tel.Log().Info(ctx, "health check completed",
				"healthy", healthy,
				"unhealthy", unhealthy,
			)

			stop()
		}
	}()
}
