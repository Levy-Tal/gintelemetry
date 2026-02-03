package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Levy-Tal/gintelemetry"
)

func TestOrderService_ProcessOrder(t *testing.T) {
	ctx := context.Background()

	// Create test telemetry configuration
	tel, _, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-service"))
	if err != nil {
		t.Fatalf("failed to start telemetry: %v", err)
	}
	defer tel.Shutdown(ctx)

	service := NewOrderService(tel)

	tests := []struct {
		name    string
		orderID string
		wantErr bool
	}{
		{
			name:    "valid order",
			orderID: "order-123",
			wantErr: false,
		},
		{
			name:    "empty order ID",
			orderID: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ProcessOrder(ctx, tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderService_GetOrderStatus(t *testing.T) {
	ctx := context.Background()

	tel, _, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-service"))
	if err != nil {
		t.Fatalf("failed to start telemetry: %v", err)
	}
	defer tel.Shutdown(ctx)

	service := NewOrderService(tel)

	tests := []struct {
		name       string
		orderID    string
		wantStatus string
		wantErr    bool
	}{
		{
			name:       "valid order",
			orderID:    "order-123",
			wantStatus: "completed",
			wantErr:    false,
		},
		{
			name:       "invalid order",
			orderID:    "invalid",
			wantStatus: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := service.GetOrderStatus(ctx, tt.orderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrderStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if status != tt.wantStatus {
				t.Errorf("GetOrderStatus() status = %v, want %v", status, tt.wantStatus)
			}
		})
	}
}

func TestHandlers(t *testing.T) {
	ctx := context.Background()

	// Initialize telemetry for testing
	tel, router, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-handlers"))
	if err != nil {
		t.Fatalf("failed to start telemetry: %v", err)
	}
	defer tel.Shutdown(ctx)

	// Setup routes
	SetupRoutes(router, tel)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "health check",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "process valid order",
			method:         "POST",
			path:           "/orders/123/process",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "process empty order",
			method:         "POST",
			path:           "/orders//process",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get order status",
			method:         "GET",
			path:           "/orders/123/status",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get invalid order status",
			method:         "GET",
			path:           "/orders/invalid/status",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					w.Code, tt.expectedStatus)
			}
		})
	}
}

// TestWithCustomContext demonstrates testing with a custom context
func TestWithCustomContext(t *testing.T) {
	ctx := context.Background()

	tel, _, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-context"))
	if err != nil {
		t.Fatalf("failed to start telemetry: %v", err)
	}
	defer tel.Shutdown(ctx)

	// Create a custom context with a span
	ctx, stop := tel.Trace().StartSpan(ctx, "test.span")
	defer stop()

	// Add some attributes to the span
	tel.Trace().SetAttributes(ctx,
		tel.Attr().String("test.name", "custom context test"),
	)

	// Use the service with the custom context
	service := NewOrderService(tel)
	err = service.ProcessOrder(ctx, "test-order")
	if err != nil {
		t.Errorf("ProcessOrder failed: %v", err)
	}
}

// TestConcurrentRequests demonstrates testing concurrent operations
func TestConcurrentRequests(t *testing.T) {
	ctx := context.Background()

	tel, router, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-concurrent"))
	if err != nil {
		t.Fatalf("failed to start telemetry: %v", err)
	}
	defer tel.Shutdown(ctx)

	SetupRoutes(router, tel)

	// Run multiple concurrent requests
	const numRequests = 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(i int) {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("request %d failed: got status %v", i, w.Code)
			}
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// BenchmarkProcessOrder shows how to benchmark instrumented code
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
