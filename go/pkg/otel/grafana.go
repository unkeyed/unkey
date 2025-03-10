package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"go.opentelemetry.io/contrib/instrumentation/runtime"

	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/metric"

	"go.opentelemetry.io/otel/sdk/trace"
)

// Config defines the configuration settings for OpenTelemetry integration with Grafana.
// It specifies connection details and application metadata needed for proper telemetry.
type Config struct {
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
// The function returns a slice of shutdown functions that should be called in reverse
// order during application shutdown to ensure proper cleanup of telemetry resources.
//
// Example:
//
//	shutdownFuncs, err := otel.InitGrafana(ctx, otel.Config{
//	    GrafanaEndpoint: "https://otlp-gateway-prod-us-east-0.grafana.net/otlp",
//	    Application:     "unkey-api",
//	    Version:         version.Version,
//	})
//	if err != nil {
//	    log.Fatalf("Failed to initialize telemetry: %v", err)
//	}
//
//	// Later during shutdown:
//	for i := len(shutdownFuncs) - 1; i >= 0; i-- {
//	    if err := shutdownFuncs[i](ctx); err != nil {
//	        log.Printf("Error during telemetry shutdown: %v", err)
//	    }
//	}
func InitGrafana(ctx context.Context, config Config) ([]shutdown.ShutdownFn, error) {
	shutdowns := make([]shutdown.ShutdownFn, 0)

	// Initialize trace exporter
	traceExporter, err := otlptrace.New(ctx, otlptracehttp.NewClient())
	if err != nil {
		return nil, fmt.Errorf("unable to init grafana tracing: %w", err)
	}
	shutdowns = append(shutdowns, traceExporter.Shutdown)

	// Create and register trace provider
	traceProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter))
	shutdowns = append(shutdowns, traceProvider.Shutdown)

	tracing.SetGlobalTraceProvider(traceProvider)

	// Initialize metrics exporter
	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpoint(config.GrafanaEndpoint))
	if err != nil {
		return nil, fmt.Errorf("unable to init grafana metrics: %w", err)
	}
	shutdowns = append(shutdowns, metricExporter.Shutdown)

	// Create and register meter provider
	meterProvider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(metricExporter)))
	shutdowns = append(shutdowns, meterProvider.Shutdown)

	// Initialize application metrics
	err = metrics.Init(meterProvider.Meter(config.Application))
	if err != nil {
		return nil, fmt.Errorf("unable to init custom metrics: %w", err)
	}

	// Collect runtime metrics (memory, GC, goroutines, etc.)
	err = runtime.Start(
		runtime.WithMeterProvider(meterProvider),
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to init runtime metrics: %w", err)
	}

	return shutdowns, nil
}
