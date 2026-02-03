package gintelemetry

import (
	"fmt"
	"os"
	"time"

	"github.com/Levy-Tal/gintelemetry/internal/exporter"
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
	// Defaults to OTEL_EXPORTER_OTLP_ENDPOINT environment variable.
	Endpoint string

	// Protocol specifies the transport protocol (grpc or http).
	// Defaults to ProtocolGRPC.
	Protocol Protocol

	// Insecure determines whether to use a non-TLS connection.
	// Defaults to true if not explicitly set.
	// Set to false to enable TLS.
	Insecure bool

	// TLS configuration for secure connections (when Insecure = false).
	TLS *TLSConfig

	// LogLevel sets the minimum log level. Defaults to LevelInfo.
	// Use LevelDebug, LevelInfo, LevelWarn, LevelError.
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

	// ExporterRetries specifies the number of retry attempts when creating exporters.
	// This is useful in environments like Kubernetes where the collector may not be
	// immediately available. Set to 0 for no retries (fail fast), or a positive number
	// for retry attempts with exponential backoff. Defaults to 3 retries.
	ExporterRetries int
}

type TLSConfig struct {
	CertFile           string
	KeyFile            string
	CAFile             string
	InsecureSkipVerify bool
}

func (c Config) copy() Config {
	cfgCopy := Config{
		ServiceName:       c.ServiceName,
		Endpoint:          c.Endpoint,
		Protocol:          c.Protocol,
		Insecure:          c.Insecure,
		LogLevel:          c.LogLevel,
		ShutdownTimeout:   c.ShutdownTimeout,
		SetGlobalProvider: c.SetGlobalProvider,
		ExporterRetries:   c.ExporterRetries,
	}
	if c.TLS != nil {
		cfgCopy.TLS = &TLSConfig{
			CertFile:           c.TLS.CertFile,
			KeyFile:            c.TLS.KeyFile,
			CAFile:             c.TLS.CAFile,
			InsecureSkipVerify: c.TLS.InsecureSkipVerify,
		}
	}
	if c.GlobalAttributes != nil {
		cfgCopy.GlobalAttributes = make(map[string]string, len(c.GlobalAttributes))
		for k, v := range c.GlobalAttributes {
			cfgCopy.GlobalAttributes[k] = v
		}
	}
	return cfgCopy
}

func (c Config) WithInsecure(insecure bool) Config {
	c.Insecure = insecure
	return c
}

func (c Config) WithLogLevel(level Level) Config {
	c.LogLevel = level
	return c
}

func (c Config) WithTLS(tls *TLSConfig) Config {
	if tls != nil {
		c.TLS = &TLSConfig{
			CertFile:           tls.CertFile,
			KeyFile:            tls.KeyFile,
			CAFile:             tls.CAFile,
			InsecureSkipVerify: tls.InsecureSkipVerify,
		}
	}
	c.Insecure = false
	return c
}

func (c Config) WithMTLS(certFile, keyFile, caFile string) Config {
	c.TLS = &TLSConfig{CertFile: certFile, KeyFile: keyFile, CAFile: caFile}
	c.Insecure = false
	return c
}

func (c Config) WithTrustedCA(caFile string) Config {
	c.TLS = &TLSConfig{CAFile: caFile}
	c.Insecure = false
	return c
}

func (c Config) WithProtocol(protocol Protocol) Config {
	c.Protocol = protocol
	return c
}

func (c Config) WithHTTP() Config {
	c.Protocol = ProtocolHTTP
	return c
}

func (c Config) WithGRPC() Config {
	c.Protocol = ProtocolGRPC
	return c
}

func (c Config) WithGlobalAttributes(attrs map[string]string) Config {
	if c.GlobalAttributes == nil {
		c.GlobalAttributes = make(map[string]string, len(attrs))
	}
	for k, v := range attrs {
		c.GlobalAttributes[k] = v
	}
	return c
}

func (c Config) WithGlobalAttribute(key, value string) Config {
	if c.GlobalAttributes == nil {
		c.GlobalAttributes = make(map[string]string)
	}
	c.GlobalAttributes[key] = value
	return c
}

type validatedConfig struct {
	Config
	endpoint string
}

func (c *Config) validate() (*validatedConfig, error) {
	// Check OTEL_SERVICE_NAME if ServiceName not set
	if c.ServiceName == "" {
		c.ServiceName = os.Getenv("OTEL_SERVICE_NAME")
	}
	if c.ServiceName == "" {
		return nil, fmt.Errorf("gintelemetry: ServiceName must be provided (via Config.ServiceName or OTEL_SERVICE_NAME env var)")
	}

	// Check OTEL_EXPORTER_OTLP_ENDPOINT if Endpoint not set
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("gintelemetry: Endpoint must be provided (via Config.Endpoint or OTEL_EXPORTER_OTLP_ENDPOINT env var)")
	}

	// Check OTEL_EXPORTER_OTLP_PROTOCOL if Protocol not set
	if c.Protocol == "" {
		if proto := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); proto != "" {
			switch proto {
			case "grpc":
				c.Protocol = ProtocolGRPC
			case "http/protobuf", "http":
				c.Protocol = ProtocolHTTP
			default:
				return nil, fmt.Errorf("gintelemetry: invalid OTEL_EXPORTER_OTLP_PROTOCOL value: %s (must be 'grpc' or 'http')", proto)
			}
		}
	}

	// Load global attributes from environment variables
	// OTEL_RESOURCE_ATTRIBUTES format: key1=value1,key2=value2
	if resourceAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES"); resourceAttrs != "" {
		if c.GlobalAttributes == nil {
			c.GlobalAttributes = make(map[string]string)
		}
		// Parse comma-separated key=value pairs
		pairs := splitResourceAttributes(resourceAttrs)
		for _, pair := range pairs {
			if key, value, ok := parseKeyValue(pair); ok {
				// Config attributes take precedence over env vars
				if _, exists := c.GlobalAttributes[key]; !exists {
					c.GlobalAttributes[key] = value
				}
			}
		}
	}

	return &validatedConfig{Config: *c, endpoint: endpoint}, nil
}

// splitResourceAttributes splits comma-separated attributes, handling escaped commas
func splitResourceAttributes(s string) []string {
	var result []string
	var current string
	escaped := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			// Handle escape sequences
			escaped = true
			i++
			current += string(s[i])
		} else if s[i] == ',' && !escaped {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(s[i])
			escaped = false
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// parseKeyValue parses a key=value pair, handling escaped equals signs
func parseKeyValue(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++ // skip escaped character
			continue
		}
		if s[i] == '=' {
			key := s[:i]
			value := s[i+1:]
			// Trim whitespace
			key = trimString(key)
			value = trimString(value)
			if key != "" {
				return key, value, true
			}
			return "", "", false
		}
	}
	return "", "", false
}

// trimString removes leading and trailing whitespace
func trimString(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}

func (vc *validatedConfig) buildExporterConfig() exporter.ExporterConfig {
	protocol := vc.Protocol
	if protocol == "" {
		protocol = ProtocolGRPC
	}

	var tlsCfg *exporter.TLSConfig
	if vc.TLS != nil {
		tlsCfg = &exporter.TLSConfig{
			CertFile:           vc.TLS.CertFile,
			KeyFile:            vc.TLS.KeyFile,
			CAFile:             vc.TLS.CAFile,
			InsecureSkipVerify: vc.TLS.InsecureSkipVerify,
		}
	}

	return exporter.ExporterConfig{
		Endpoint: vc.endpoint,
		Protocol: exporter.Protocol(protocol),
		Insecure: vc.Insecure,
		TLS:      tlsCfg,
	}
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

func (c *Config) getExporterRetries() int {
	if c.ExporterRetries > 0 {
		return c.ExporterRetries
	}
	return 3
}

func (c Config) WithExporterRetries(retries int) Config {
	c.ExporterRetries = retries
	return c
}

func (c Config) WithNoRetry() Config {
	c.ExporterRetries = 0
	return c
}
