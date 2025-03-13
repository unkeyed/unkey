package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"go.opentelemetry.io/contrib/instrumentation/runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Config defines the configuration settings for OpenTelemetry integration with Grafana.
// It specifies connection details and application metadata needed for proper telemetry.
type Config struct {
	// NodeID is a unique identifier for the current service instance,
	// used to distinguish between multiple instances of the same service.
	NodeID string

	// CloudRegion indicates the geographic region where this service instance is running,
	// which helps with identifying regional performance patterns or issues.
	CloudRegion string

	// GrafanaEndpoint is the URL endpoint where telemetry data will be sent.
	// For Grafana Cloud, this looks like "https://otlp-gateway-{your-stack-id}.grafana.net/otlp"
	GrafanaEndpoint string

	// Application is the name of your application, used to identify the source of telemetry data.
	// This appears in Grafana dashboards and alerts.
	Application string

	// Version is the current version of your application, allowing you to correlate
	// behavior changes with specific releases.
	Version string
}

// InitGrafana initializes the global tracer and metric providers for OpenTelemetry,
// configured to send telemetry data to Grafana Cloud or any compatible OTLP endpoint.
//
// It sets up:
// - Distributed tracing using the OTLP HTTP exporter
// - Metrics collection via OTLP HTTP exporter
// - Runtime metrics for Go applications (memory, GC, goroutines, etc.)
// - Custom application metrics defined in the metrics package
//
// The function registers all necessary shutdown handlers with the provided shutdowns instance.
// These handlers will be called during application termination to ensure proper cleanup.
//
// Example:
//
//	shutdowns := shutdown.New()
//	err := otel.InitGrafana(ctx, otel.Config{
//	    GrafanaEndpoint: "https://otlp-gateway-prod-us-east-0.grafana.net/otlp",
//	    Application:     "unkey-api",
//	    Version:         version.Version,
//	}, shutdowns)
//
//	if err != nil {
//	    log.Fatalf("Failed to initialize telemetry: %v", err)
//	}
//
//	// Later during shutdown:
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	errs := shutdowns.Shutdown(ctx)
//	for _, err := range errs {
//	    log.Printf("Shutdown error: %v", err)
//	}
func InitGrafana(ctx context.Context, config Config, shutdowns *shutdown.Shutdowns) error {
	// Create a resource with common attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.Application),
			semconv.ServiceVersion(config.Version),
			semconv.ServiceInstanceID(config.NodeID),
			semconv.CloudRegion(config.CloudRegion),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize trace exporter with configuration matching the old implementation
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(config.GrafanaEndpoint),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithInsecure(), // For local development

	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register shutdown function for trace exporter
	shutdowns.RegisterCtx(traceExporter.Shutdown)

	// Create and register trace provider with the same batch settings as the old code
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)

	// Register shutdown function for trace provider
	shutdowns.RegisterCtx(traceProvider.Shutdown)

	// Set the global trace provider
	otel.SetTracerProvider(traceProvider)
	tracing.SetGlobalTraceProvider(traceProvider)

	// Initialize metrics exporter with configuration matching the old implementation
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(config.GrafanaEndpoint),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		otlpmetrichttp.WithInsecure(), // For local development

	)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Register shutdown function for metric exporter
	shutdowns.RegisterCtx(metricExporter.Shutdown)

	// Create and register meter provider with the same reader settings as the old code
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(
			metric.NewPeriodicReader(
				metricExporter,
				metric.WithInterval(10*time.Second), // Match the 10s interval from the old code
			),
		),
		metric.WithResource(res),
	)

	// Register shutdown function for meter provider
	shutdowns.RegisterCtx(meterProvider.Shutdown)

	// Set the global meter provider
	otel.SetMeterProvider(meterProvider)

	// Initialize application metrics
	err = metrics.Init(meterProvider.Meter(config.Application))
	if err != nil {
		return fmt.Errorf("failed to initialize custom metrics: %w", err)
	}

	// Collect runtime metrics (memory, GC, goroutines, etc.)
	err = runtime.Start(
		runtime.WithMeterProvider(meterProvider),
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to start runtime metrics collection: %w", err)
	}

	return nil
}
