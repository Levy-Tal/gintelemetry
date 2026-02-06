package gintelemetry

import (
	"fmt"
	"os"
	"time"
)

// Protocol defines the transport protocol for OTLP exporters.
type Protocol string

const (
	// ProtocolGRPC uses gRPC for OTLP transport (default).
	ProtocolGRPC Protocol = "grpc"

	// ProtocolHTTP uses HTTP/protobuf for OTLP transport.
	ProtocolHTTP Protocol = "http"
)

// Config holds the configuration for initializing the telemetry stack.
type Config struct {
	// ServiceName is required and reported in all telemetry data.
	ServiceName string

	// Endpoint is the OTLP collector endpoint.
	// For gRPC: "localhost:4317" (default port 4317)
	// For HTTP: "localhost:4318" (default port 4318)
	Endpoint string

	// Protocol specifies the transport protocol (grpc or http).
	// Defaults to ProtocolGRPC.
	Protocol Protocol

	// Insecure determines whether to use a non-TLS connection.
	// Defaults to true for local development.
	Insecure bool

	// LogLevel sets the minimum log level. Defaults to LevelInfo.
	LogLevel Level

	// GlobalAttributes are added to all telemetry (traces, metrics, logs).
	// Use this for team names, environment, region, etc.
	GlobalAttributes map[string]string

	// ShutdownTimeout is the maximum time to wait for telemetry shutdown.
	// Defaults to 10 seconds if not set.
	ShutdownTimeout time.Duration

	// SetGlobalProvider controls whether to set the global OpenTelemetry provider.
	// WARNING: Setting this to true makes the telemetry system use global state,
	// which can cause issues with concurrent tests and multiple service instances.
	// Only enable this if you rely on instrumentation libraries that require the
	// global provider. The default (false) provides isolated, instance-based telemetry.
	SetGlobalProvider bool
}

func (c *Config) validate() error {
	// Check OTEL_SERVICE_NAME if ServiceName not set
	if c.ServiceName == "" {
		c.ServiceName = os.Getenv("OTEL_SERVICE_NAME")
	}
	if c.ServiceName == "" {
		return fmt.Errorf("gintelemetry: ServiceName is required")
	}

	// Check OTEL_EXPORTER_OTLP_ENDPOINT if Endpoint not set
	if c.Endpoint == "" {
		c.Endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if c.Endpoint == "" {
		return fmt.Errorf("gintelemetry: Endpoint is required")
	}

	// Set default protocol if not specified
	if c.Protocol == "" {
		c.Protocol = ProtocolGRPC
	}

	return nil
}

func (c *Config) getLogLevel() Level {
	if c.LogLevel != 0 {
		return c.LogLevel
	}
	return LevelInfo
}

func (c *Config) getShutdownTimeout() time.Duration {
	if c.ShutdownTimeout > 0 {
		return c.ShutdownTimeout
	}
	return 10 * time.Second
}
