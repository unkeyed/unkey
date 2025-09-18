package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/assetmanagerd/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// InitProviders initializes OpenTelemetry providers
func InitProviders(ctx context.Context, cfg *config.Config, version string) (func(context.Context) error, error) {
	// AIDEV-NOTE: Dynamic version injection for unified telemetry
	// Schema conflict fix - Using semconv v1.26.0
	res, err := resource.New(ctx,
		resource.WithAttributes(
			ServiceAttributes(cfg.OTELServiceName, version)...,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize trace provider
	traceProvider, err := initTraceProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize trace provider: %w", err)
	}

	// Initialize metric provider
	metricProvider, err := initMetricProvider(ctx, cfg, res)
	if err != nil {
		_ = traceProvider.Shutdown(ctx)
		return nil, fmt.Errorf("failed to initialize metric provider: %w", err)
	}

	// Set global providers
	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(metricProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown function
	return func(ctx context.Context) error {
		err := traceProvider.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("failed to shutdown trace provider: %w", err)
		}

		err = metricProvider.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("failed to shutdown metric provider: %w", err)
		}

		return nil
	}, nil
}

// initTraceProvider initializes the trace provider
func initTraceProvider(ctx context.Context, cfg *config.Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.OTELEndpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.OTELSamplingRate)),
	)

	return provider, nil
}

// initMetricProvider initializes the metric provider
func initMetricProvider(ctx context.Context, cfg *config.Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	var readers []sdkmetric.Reader

	// OTLP metric exporter
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(cfg.OTELEndpoint),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	readers = append(readers, sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(10*time.Second),
	))

	// Prometheus exporter
	if cfg.OTELPrometheusEnabled {
		promExporter, err := prometheus.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
		}
		readers = append(readers, promExporter)
	}

	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	for _, reader := range readers {
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	provider := sdkmetric.NewMeterProvider(opts...)

	return provider, nil
}

// ServiceAttributes returns OTEL resource attributes for the service
func ServiceAttributes(serviceName, version string) []attribute.KeyValue {
	// AIDEV-NOTE: Dynamic version parameter for unified telemetry
	return []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(version),
		attribute.String("service.namespace", "unkey"),
		attribute.String("service.instance.id", serviceName),
	}
}

// NewMetricsServer creates a new HTTP server for Prometheus metrics
func NewMetricsServer(addr string, healthHandler http.HandlerFunc) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The prometheus handler is registered globally
		http.DefaultServeMux.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/health", healthHandler)

	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// GetMeter returns a meter for the given name
func GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}
