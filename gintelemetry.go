// Package gintelemetry provides opinionated OpenTelemetry bootstrap for Gin applications.
package gintelemetry

import (
	"context"
	"fmt"
	"os"

	"github.com/Levy-Tal/gintelemetry/internal/exporter"
	"github.com/Levy-Tal/gintelemetry/internal/provider"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// Start initializes the telemetry stack and returns a configured Gin router.
func Start(ctx context.Context, cfg Config) (*gin.Engine, func(context.Context) error, error) {
	if cfg.ServiceName == "" {
		return nil, nil, fmt.Errorf("gintelemetry: ServiceName must be provided")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if endpoint == "" {
		return nil, nil, fmt.Errorf("gintelemetry: OTEL_EXPORTER_OTLP_ENDPOINT not set and Config.Endpoint is empty")
	}

	insecure := cfg.Insecure
	if !cfg.insecureSet {
		insecure = true
	}

	protocol := cfg.Protocol
	if !cfg.protocolSet {
		protocol = ProtocolGRPC
	}

	var tlsCfg *exporter.TLSConfig
	if cfg.TLS != nil {
		tlsCfg = &exporter.TLSConfig{
			CertFile:           cfg.TLS.CertFile,
			KeyFile:            cfg.TLS.KeyFile,
			CAFile:             cfg.TLS.CAFile,
			InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
		}
	}

	exporterCfg := exporter.ExporterConfig{
		Endpoint: endpoint,
		Protocol: exporter.Protocol(protocol),
		Insecure: insecure,
		TLS:      tlsCfg,
	}

	traceExporter, err := exporter.NewTraceExporter(ctx, exporterCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	metricExporter, err := exporter.NewMetricExporter(ctx, exporterCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	logExporter, err := exporter.NewLogExporter(ctx, exporterCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	providerCfg := provider.ProviderConfig{
		ServiceName:      cfg.ServiceName,
		GlobalAttributes: cfg.GlobalAttributes,
	}

	res, err := provider.NewResource(providerCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	providers := &provider.Providers{
		TracerProvider: provider.NewTracerProvider(ctx, traceExporter, res),
		MeterProvider:  provider.NewMeterProvider(ctx, metricExporter, res),
		LoggerProvider: provider.NewLoggerProvider(ctx, logExporter, res),
	}

	logLevel := cfg.LogLevel
	if !cfg.logLevelSet {
		logLevel = LevelInfo
	}

	logger := otelslog.NewLogger(cfg.ServiceName, otelslog.WithLoggerProvider(providers.LoggerProvider))
	setLogger(logger, logLevel)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(otelgin.Middleware(cfg.ServiceName))

	shutdown := func(shutdownCtx context.Context) error {
		return providers.Shutdown(shutdownCtx)
	}

	return router, shutdown, nil
}
