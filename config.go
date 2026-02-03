package gintelemetry

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

	insecureSet bool
	logLevelSet bool
	protocolSet bool
}

// TLSConfig holds TLS configuration for secure gRPC connections.
type TLSConfig struct {
	// CertFile is the path to the client certificate file (for mTLS).
	// Optional - only needed for mutual TLS authentication.
	CertFile string

	// KeyFile is the path to the client private key file (for mTLS).
	// Optional - only needed for mutual TLS authentication.
	KeyFile string

	// CAFile is the path to the CA certificate file for verifying the server.
	// Optional - if not provided, system CA pool is used.
	CAFile string

	// InsecureSkipVerify skips server certificate verification.
	// WARNING: Only use for testing! Do not use in production.
	InsecureSkipVerify bool
}

// WithInsecure explicitly sets the Insecure flag.
func (c Config) WithInsecure(insecure bool) Config {
	c.Insecure = insecure
	c.insecureSet = true
	return c
}

// WithLogLevel explicitly sets the log level.
func (c Config) WithLogLevel(level Level) Config {
	c.LogLevel = level
	c.logLevelSet = true
	return c
}

// WithTLS sets the TLS configuration for secure connections.
// Automatically sets Insecure to false.
func (c Config) WithTLS(tls *TLSConfig) Config {
	c.TLS = tls
	c.Insecure = false
	c.insecureSet = true
	return c
}

// WithMTLS configures mutual TLS authentication.
// Automatically sets Insecure to false.
func (c Config) WithMTLS(certFile, keyFile, caFile string) Config {
	c.TLS = &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
	}
	c.Insecure = false
	c.insecureSet = true
	return c
}

// WithTrustedCA configures TLS with a trusted CA certificate.
// Automatically sets Insecure to false.
func (c Config) WithTrustedCA(caFile string) Config {
	c.TLS = &TLSConfig{
		CAFile: caFile,
	}
	c.Insecure = false
	c.insecureSet = true
	return c
}

// WithProtocol explicitly sets the transport protocol (grpc or http).
func (c Config) WithProtocol(protocol Protocol) Config {
	c.Protocol = protocol
	c.protocolSet = true
	return c
}

// WithHTTP is a convenience method to use HTTP protocol.
func (c Config) WithHTTP() Config {
	c.Protocol = ProtocolHTTP
	c.protocolSet = true
	return c
}

// WithGRPC is a convenience method to use gRPC protocol.
func (c Config) WithGRPC() Config {
	c.Protocol = ProtocolGRPC
	c.protocolSet = true
	return c
}

// WithGlobalAttributes sets global attributes for all telemetry.
func (c Config) WithGlobalAttributes(attrs map[string]string) Config {
	c.GlobalAttributes = attrs
	return c
}

// WithGlobalAttribute adds a single global attribute.
func (c Config) WithGlobalAttribute(key, value string) Config {
	if c.GlobalAttributes == nil {
		c.GlobalAttributes = make(map[string]string)
	}
	c.GlobalAttributes[key] = value
	return c
}
