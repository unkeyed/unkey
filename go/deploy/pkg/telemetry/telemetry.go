// Package telemetry provides OpenTelemetry instrumentation for distributed tracing and metrics collection.
//
// This package initializes and manages OpenTelemetry providers for both tracing and metrics,
// supporting OTLP export and Prometheus metrics exposure. It handles resource creation,
// provider lifecycle management, and HTTP handler instrumentation.
//
// Basic usage:
//
//	cfg := &telemetry.Config{
//		Enabled:             true,
//		ServiceName:         "my-service",
//		ServiceVersion:      "1.0.0",
//		TracingSamplingRate: 1.0,
//		OTLPEndpoint:        "http://localhost:4318",
//		PrometheusEnabled:   true,
//		PrometheusPort:      "9090",
//	}
//
//	provider, err := telemetry.Initialize(ctx, cfg, logger)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer provider.Shutdown(ctx)
//
//	// Wrap HTTP handlers for automatic instrumentation
//	handler := provider.WrapHandler(myHandler, "operation-name")
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds telemetry configuration for OpenTelemetry providers.
//
// All configuration options are optional and have sensible defaults.
// TracingSamplingRate should be between 0.0 and 1.0, where 1.0 means
// all traces are sampled.
type Config struct {
	// Enabled controls whether telemetry is active. When false, Initialize returns a no-op provider.
	Enabled bool
	// ServiceName identifies the service in telemetry data and should be consistent across instances.
	ServiceName string
	// ServiceVersion is included in telemetry resource attributes for version tracking.
	ServiceVersion string
	// TracingSamplingRate controls the fraction of traces to sample (0.0 to 1.0).
	TracingSamplingRate float64
	// OTLPEndpoint is the HTTP endpoint for OTLP trace export.
	OTLPEndpoint string
	// PrometheusEnabled controls whether metrics are exposed via Prometheus HTTP endpoint.
	PrometheusEnabled bool
	// PrometheusPort specifies the port for the Prometheus metrics server.
	PrometheusPort string
	// PrometheusInterface specifies the network interface for the Prometheus metrics server.
	PrometheusInterface string
	// HighCardinalityLabelsEnabled allows high-cardinality metric labels when true.
	HighCardinalityLabelsEnabled bool
}

// Provider holds initialized OpenTelemetry providers and manages their lifecycle.
//
// Provider is safe for concurrent use and handles graceful shutdown of all
// telemetry components. When telemetry is disabled, Provider methods are safe
// to call but perform no operations.
type Provider struct {
	// TracerProvider provides distributed tracing capabilities via OpenTelemetry.
	TracerProvider *sdktrace.TracerProvider
	// MeterProvider provides metrics collection capabilities via OpenTelemetry.
	MeterProvider *sdkmetric.MeterProvider
	// PrometheusHTTP serves Prometheus metrics when metrics are enabled.
	PrometheusHTTP http.Handler
	// Resource contains service identification attributes used by all providers.
	Resource      *resource.Resource
	promExporter  *prometheus.Exporter
	shutdownFuncs []func(context.Context) error
	mu            sync.Mutex
}

// Initialize sets up OpenTelemetry providers for tracing and metrics collection.
//
// When cfg.Enabled is false, Initialize returns a no-op Provider that is safe to use
// but performs no telemetry operations. The returned Provider must be shut down
// via Shutdown to ensure proper cleanup of resources.
//
// Initialize returns an error if provider creation fails, OTLP endpoint is unreachable,
// or resource initialization encounters issues.
func Initialize(ctx context.Context, cfg *Config, logger *slog.Logger) (*Provider, error) {
	if !cfg.Enabled {
		logger.Info("OpenTelemetry disabled")
		return &Provider{}, nil
	}

	// Create resource with service information
	// AIDEV-NOTE: Use resource.New() instead of resource.Merge() to avoid schema conflicts
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.ServiceInstanceID(getInstanceID()),
		),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	provider := &Provider{
		Resource:      res,
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	// Initialize tracing
	if err := provider.initTracing(ctx, cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}

	// Initialize metrics
	if err := provider.initMetrics(ctx, cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Set global propagator
	otel.SetTextMapPropagator(propagation.TraceContext{})

	logger.Info("OpenTelemetry initialized",
		slog.String("service_name", cfg.ServiceName),
		slog.String("service_version", cfg.ServiceVersion),
		slog.String("endpoint", cfg.OTLPEndpoint),
		slog.Bool("prometheus_enabled", cfg.PrometheusEnabled),
	)

	return provider, nil
}

func (p *Provider) initTracing(ctx context.Context, cfg *Config, logger *slog.Logger) error {
	exporter, err := otlptrace.New(ctx,
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
			otlptracehttp.WithInsecure(),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(p.Resource),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.TracingSamplingRate)),
	)

	otel.SetTracerProvider(tp)
	p.TracerProvider = tp
	p.addShutdownFunc(tp.Shutdown)

	return nil
}

func (p *Provider) initMetrics(ctx context.Context, cfg *Config, logger *slog.Logger) error {
	var readers []sdkmetric.Reader

	// Add Prometheus exporter if enabled
	if cfg.PrometheusEnabled {
		promExporter, err := prometheus.New()
		if err != nil {
			return fmt.Errorf("failed to create prometheus exporter: %w", err)
		}
		readers = append(readers, promExporter)
	}

	// Create meter provider with readers
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(p.Resource),
	}
	for _, reader := range readers {
		opts = append(opts, sdkmetric.WithReader(reader))
	}
	mp := sdkmetric.NewMeterProvider(opts...)

	otel.SetMeterProvider(mp)
	p.MeterProvider = mp
	p.addShutdownFunc(mp.Shutdown)

	// Set up Prometheus HTTP handler if enabled
	if cfg.PrometheusEnabled {
		// The prometheus exporter automatically registers collectors with the default registry
		p.PrometheusHTTP = promhttp.Handler()
	}

	return nil
}

// Shutdown gracefully shuts down all telemetry providers and exporters.
//
// Shutdown should be called when the application terminates to ensure proper
// cleanup and flushing of any pending telemetry data. It returns the first
// error encountered during shutdown, but continues attempting to shut down
// all providers even if some fail.
func (p *Provider) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	for _, fn := range p.shutdownFuncs {
		if err := fn(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// WrapHandler wraps an HTTP handler with OpenTelemetry tracing instrumentation.
//
// The operation parameter is used as the span name for requests handled by this handler.
// When tracing is disabled, WrapHandler returns the original handler unchanged.
func (p *Provider) WrapHandler(handler http.Handler, operation string) http.Handler {
	if p.TracerProvider == nil {
		return handler
	}
	return otelhttp.NewHandler(handler, operation)
}

func (p *Provider) addShutdownFunc(fn func(context.Context) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shutdownFuncs = append(p.shutdownFuncs, fn)
}

func getInstanceID() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
