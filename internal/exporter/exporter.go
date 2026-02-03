package exporter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

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

type Protocol string

const (
	ProtocolGRPC Protocol = "grpc"
	ProtocolHTTP Protocol = "http"
)

type ExporterConfig struct {
	Endpoint string
	Protocol Protocol
	Insecure bool
	TLS      *TLSConfig
}

type TLSConfig struct {
	CertFile           string
	KeyFile            string
	CAFile             string
	InsecureSkipVerify bool
}

func buildTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	if cfg == nil {
		return nil, nil
	}
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},
	}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
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

func newHTTPClient(tlsConfig *tls.Config) *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			MaxConnsPerHost:       100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			TLSClientConfig:       tlsConfig,
		},
	}
}

func retryWithBackoff(ctx context.Context, maxRetries int, operation func(context.Context) error) error {
	if maxRetries <= 0 {
		maxRetries = 1
	}
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled before attempt %d: %w", attempt+1, err)
		}
		if err := operation(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt < maxRetries-1 {
			backoff := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
			if backoff > 1600*time.Millisecond {
				backoff = 1600 * time.Millisecond
			}
			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			}
		}
	}
	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func NewTraceExporter(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	if cfg.Protocol == ProtocolHTTP {
		return newTraceExporterHTTP(ctx, cfg)
	}
	return newTraceExporterGRPC(ctx, cfg)
}

func NewTraceExporterWithRetry(ctx context.Context, cfg ExporterConfig, maxRetries int) (sdktrace.SpanExporter, error) {
	var exporter sdktrace.SpanExporter
	err := retryWithBackoff(ctx, maxRetries, func(ctx context.Context) error {
		var err error
		exporter, err = NewTraceExporter(ctx, cfg)
		return err
	})
	return exporter, err
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
		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig.Clone())))
	}
	return otlptracegrpc.New(ctx, opts...)
}

func newTraceExporterHTTP(ctx context.Context, cfg ExporterConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure(), otlptracehttp.WithHTTPClient(newHTTPClient(nil)))
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlptracehttp.WithHTTPClient(newHTTPClient(tlsConfig.Clone())))
	} else {
		opts = append(opts, otlptracehttp.WithHTTPClient(newHTTPClient(nil)))
	}
	return otlptracehttp.New(ctx, opts...)
}

func NewMetricExporter(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	if cfg.Protocol == ProtocolHTTP {
		return newMetricExporterHTTP(ctx, cfg)
	}
	return newMetricExporterGRPC(ctx, cfg)
}

func NewMetricExporterWithRetry(ctx context.Context, cfg ExporterConfig, maxRetries int) (sdkmetric.Exporter, error) {
	var exporter sdkmetric.Exporter
	err := retryWithBackoff(ctx, maxRetries, func(ctx context.Context) error {
		var err error
		exporter, err = NewMetricExporter(ctx, cfg)
		return err
	})
	return exporter, err
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
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig.Clone())))
	}
	return otlpmetricgrpc.New(ctx, opts...)
}

func newMetricExporterHTTP(ctx context.Context, cfg ExporterConfig) (sdkmetric.Exporter, error) {
	opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure(), otlpmetrichttp.WithHTTPClient(newHTTPClient(nil)))
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlpmetrichttp.WithHTTPClient(newHTTPClient(tlsConfig.Clone())))
	} else {
		opts = append(opts, otlpmetrichttp.WithHTTPClient(newHTTPClient(nil)))
	}
	return otlpmetrichttp.New(ctx, opts...)
}

func NewLogExporter(ctx context.Context, cfg ExporterConfig) (sdklog.Exporter, error) {
	if cfg.Protocol == ProtocolHTTP {
		return newLogExporterHTTP(ctx, cfg)
	}
	return newLogExporterGRPC(ctx, cfg)
}

func NewLogExporterWithRetry(ctx context.Context, cfg ExporterConfig, maxRetries int) (sdklog.Exporter, error) {
	var exporter sdklog.Exporter
	err := retryWithBackoff(ctx, maxRetries, func(ctx context.Context) error {
		var err error
		exporter, err = NewLogExporter(ctx, cfg)
		return err
	})
	return exporter, err
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
		opts = append(opts, otlploggrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig.Clone())))
	}
	return otlploggrpc.New(ctx, opts...)
}

func newLogExporterHTTP(ctx context.Context, cfg ExporterConfig) (sdklog.Exporter, error) {
	opts := []otlploghttp.Option{otlploghttp.WithEndpoint(cfg.Endpoint)}
	if cfg.Insecure {
		opts = append(opts, otlploghttp.WithInsecure(), otlploghttp.WithHTTPClient(newHTTPClient(nil)))
	} else if cfg.TLS != nil {
		tlsConfig, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		opts = append(opts, otlploghttp.WithHTTPClient(newHTTPClient(tlsConfig.Clone())))
	} else {
		opts = append(opts, otlploghttp.WithHTTPClient(newHTTPClient(nil)))
	}
	return otlploghttp.New(ctx, opts...)
}
