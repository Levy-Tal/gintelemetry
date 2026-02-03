package exporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
)

// Protocol defines the transport protocol.
type Protocol string

const (
	ProtocolGRPC Protocol = "grpc"
	ProtocolHTTP Protocol = "http"
)

// ExporterConfig holds the configuration for OTLP exporters.
type ExporterConfig struct {
	Endpoint string
	Protocol Protocol
	Insecure bool
	TLS      *TLSConfig
}

// TLSConfig holds TLS configuration.
type TLSConfig struct {
	CertFile           string
	KeyFile            string
	CAFile             string
	InsecureSkipVerify bool
}

// buildTLSConfig creates a TLS configuration from the provided settings.
func buildTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	// Load client certificate for mTLS
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate for server verification
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

// NewTraceExporter creates a new OTLP trace exporter.
func NewTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	if cfg.Protocol == ProtocolHTTP {
		return newTraceExporterHTTP(ctx, cfg)
	}
	return newTraceExporterGRPC(ctx, cfg)
}

func newTraceExporterGRPC(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlptracegrpc.New(ctx, opts...)
}

func newTraceExporterHTTP(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}

	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfig))
	}

	return otlptracehttp.New(ctx, opts...)
}

// NewMetricExporter creates a new OTLP metric exporter.
func NewMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	if cfg.Protocol == ProtocolHTTP {
		return newMetricExporterHTTP(ctx, cfg)
	}
	return newMetricExporterGRPC(ctx, cfg)
}

func newMetricExporterGRPC(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(cfg.Endpoint)}

	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

func newMetricExporterHTTP(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Endpoint)}

	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithTLSClientConfig(tlsConfig))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

// NewLogExporter creates a new OTLP log exporter.
func NewLogExporter(ctx context.Context, cfg ExporterConfig) (sdklog.Exporter, error) {
	if cfg.Protocol == ProtocolHTTP {
		return newLogExporterHTTP(ctx, cfg)
	}
	return newLogExporterGRPC(ctx, cfg)
}

func newLogExporterGRPC(ctx context.Context, cfg ExporterConfig) (sdklog.Exporter, error) {
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(cfg.Endpoint)}

	if cfg.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlploggrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlploggrpc.New(ctx, opts...)
}

func newLogExporterHTTP(ctx context.Context, cfg ExporterConfig) (sdklog.Exporter, error) {
	opts := []otlploghttp.Option{otlploghttp.WithEndpoint(cfg.Endpoint)}

	if cfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlploghttp.WithTLSClientConfig(tlsConfig))
	}

	return otlploghttp.New(ctx, opts...)
}
