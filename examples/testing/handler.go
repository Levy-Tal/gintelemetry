package main

import (
	"context"
	"errors"
	"time"

	"github.com/Levy-Tal/gintelemetry"
	"github.com/gin-gonic/gin"
)

// OrderService demonstrates a service that uses telemetry
type OrderService struct {
	tel *gintelemetry.Telemetry
}

func NewOrderService(tel *gintelemetry.Telemetry) *OrderService {
	return &OrderService{tel: tel}
}

func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
	// Create a span for the operation
	ctx, stop := s.tel.Trace().StartSpan(ctx, "order.process")
	defer stop()

	s.tel.Log().Info(ctx, "processing order", "order_id", orderID)

	// Simulate validation
	if orderID == "" {
		err := errors.New("order ID is required")
		s.tel.Trace().RecordError(ctx, err)
		return err
	}

	// Simulate processing time with measurement
	start := time.Now()
	time.Sleep(10 * time.Millisecond)

	s.tel.Metric().RecordHistogram(ctx, "order.validation.duration", time.Since(start).Milliseconds())

	// Record metrics
	s.tel.Metric().AddCounter(ctx, "orders.processed", 1,
		s.tel.Attr().String("status", "success"),
	)

	s.tel.Log().Info(ctx, "order processed successfully", "order_id", orderID)
	return nil
}

func (s *OrderService) GetOrderStatus(ctx context.Context, orderID string) (string, error) {
	ctx, stop := s.tel.Trace().StartSpan(ctx, "order.get_status",
		s.tel.Attr().String("order.id", orderID),
	)
	defer stop()

	s.tel.Log().Debug(ctx, "fetching order status", "order_id", orderID)

	if orderID == "invalid" {
		err := errors.New("invalid order ID")
		s.tel.Trace().RecordError(ctx, err)
		return "", err
	}

	s.tel.Metric().AddCounter(ctx, "orders.status_checks", 1)

	return "completed", nil
}

// SetupRoutes configures the Gin routes with telemetry
func SetupRoutes(router *gin.Engine, tel *gintelemetry.Telemetry) {
	service := NewOrderService(tel)

	router.POST("/orders/:id/process", func(c *gin.Context) {
		ctx := c.Request.Context()
		orderID := c.Param("id")

		err := service.ProcessOrder(ctx, orderID)
		if err != nil {
			tel.Log().Error(ctx, "failed to process order", "error", err.Error())
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "processed"})
	})

	router.GET("/orders/:id/status", func(c *gin.Context) {
		ctx := c.Request.Context()
		orderID := c.Param("id")

		status, err := service.GetOrderStatus(ctx, orderID)
		if err != nil {
			tel.Log().Error(ctx, "failed to get order status", "error", err.Error())
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": status})
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})
}
