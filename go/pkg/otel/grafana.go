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

type Config struct {

	// WithEndpoint sets the target endpoint the Exporter will connect to. This
	// endpoint is specified as a host and optional port, no path or scheme should
	// be included.
	GrafanaEndpoint string

	Application string
	Version     string
}

// InitGrafana initializes the global tracer and metric providers.
// It returns a slice of ShutdownFuncs that should be called when the
// application is shutting down.
func InitGrafana(ctx context.Context, config Config) ([]shutdown.ShutdownFn, error) {
	shutdowns := make([]shutdown.ShutdownFn, 0)

	traceExporter, err := otlptrace.New(ctx, otlptracehttp.NewClient())
	if err != nil {
		return nil, fmt.Errorf("unable to init grafana tracing: %w", err)
	}
	shutdowns = append(shutdowns, traceExporter.Shutdown)

	traceProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter))
	shutdowns = append(shutdowns, traceProvider.Shutdown)

	tracing.SetGlobalTraceProvider(traceProvider)

	metricExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpoint(config.GrafanaEndpoint))
	if err != nil {
		return nil, fmt.Errorf("unable to init grafana metrics: %w", err)
	}
	shutdowns = append(shutdowns, metricExporter.Shutdown)

	meterProvider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(metricExporter)))
	shutdowns = append(shutdowns, meterProvider.Shutdown)

	err = metrics.Init(meterProvider.Meter(config.Application))
	if err != nil {
		return nil, fmt.Errorf("unable to init custom metrics: %w", err)
	}
	// Collect runtime metrics as well
	err = runtime.Start(
		runtime.WithMeterProvider(meterProvider),
		runtime.WithMinimumReadMemStatsInterval(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to init runtime metrics: %w", err)
	}

	return shutdowns, nil
}
