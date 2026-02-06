package gintelemetry

// NewTestConfig creates a configuration suitable for testing.
// This allows tests to run without requiring a real OTLP collector.
//
// The returned config uses insecure connections and points to a localhost endpoint.
//
// Example:
//
//	func TestMyHandler(t *testing.T) {
//	    ctx := context.Background()
//	    tel, router, err := gintelemetry.Start(ctx, gintelemetry.NewTestConfig("test-service"))
//	    if err != nil {
//	        t.Fatalf("failed to start telemetry: %v", err)
//	    }
//	    defer tel.Shutdown(ctx)
//
//	    // Test your handlers...
//	    router.GET("/test", myHandler)
//	}
func NewTestConfig(serviceName string) Config {
	return Config{
		ServiceName: serviceName,
		Endpoint:    "localhost:4317",
		Protocol:    ProtocolGRPC,
		Insecure:    true,
		LogLevel:    LevelError, // Reduce noise in tests
	}
}
