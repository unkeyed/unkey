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
	"go.opentelemetry.io/otel/sdk/resource"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds telemetry configuration
type Config struct {
	Enabled                      bool
	ServiceName                  string
	ServiceVersion               string
	TracingSamplingRate          float64
	OTLPEndpoint                 string
	PrometheusEnabled            bool
	PrometheusPort               string
	PrometheusInterface          string
	HighCardinalityLabelsEnabled bool
}

// Provider holds initialized telemetry providers
type Provider struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
	PrometheusHTTP http.Handler
	Resource       *resource.Resource
	promExporter   *prometheus.Exporter
	shutdownFuncs  []func(context.Context) error
	mu             sync.Mutex
}

// Initialize sets up OpenTelemetry providers
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

// Shutdown gracefully shuts down all telemetry providers
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

// WrapHandler wraps an HTTP handler with OpenTelemetry instrumentation
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
