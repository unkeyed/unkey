package otel

import (
	"context"
	"fmt"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/log"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"

	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Config defines the configuration settings for OpenTelemetry integration with Grafana.
// It specifies connection details and application metadata needed for proper telemetry.
type Config struct {
	// InstanceID is a unique identifier for the current service instance,
	// used to distinguish between multiple instances of the same service.
	InstanceID string

	// CloudRegion indicates the geographic region where this service instance is running,
	// which helps with identifying regional performance patterns or issues.
	CloudRegion string

	// Application is the name of your application, used to identify the source of telemetry data.
	// This appears in Grafana dashboards and alerts.
	Application string

	// TraceSampleRate controls what percentage of traces are sampled.
	// Values range from 0.0 to 1.0, where:
	// - 1.0 means all traces are sampled (100%)
	// - 0.25 means 25% of traces are sampled (the default if not specified)
	// - 0.0 means no traces are sampled (0%)
	//
	// As long as the sampling rate is greater than 0.0, all errors will be sampled.
	TraceSampleRate float64

	// PrometheusGatherer, when non-nil, enables pushing metrics over OTLP by
	// bridging this Prometheus registry into the OTLP exporter.
	//
	// Most services are scraped in-cluster via their /metrics endpoint, so
	// they leave this nil and rely on pull alone — pushing as well would
	// double-ship the same series. Set it only for services that cannot be
	// scraped (e.g. the API running against Grafana Cloud), where OTLP push
	// is the only way to get metrics out.
	PrometheusGatherer promclient.Gatherer
}

// InitGrafana initializes the global tracer and log providers for OpenTelemetry,
// configured to send telemetry data to Grafana Cloud or any compatible OTLP endpoint.
//
// It sets up:
// - Distributed tracing using the OTLP HTTP exporter
// - Log export using the OTLP HTTP exporter
// - Optionally, metrics push over OTLP (only when config.PrometheusGatherer is set)
//
// Metrics are normally exposed on a Prometheus pull endpoint (see pkg/prometheus)
// and scraped in-cluster, so pushing them over OTLP as well would double-ship the
// same series. The OTLP metric push is therefore opt-in via PrometheusGatherer and
// intended only for services that cannot be scraped (e.g. the API on Grafana Cloud).
//
// The function returns a shutdown function that should be registered with a runner instance.
// This shutdown function will be called during application termination to ensure proper cleanup.
//
// Example:
//
//	r := runner.New()
//	shutdown, err := otel.InitGrafana(ctx, otel.Config{
//	    GrafanaEndpoint: "https://otlp-sentinel-prod-us-east-0.grafana.net/otlp",
//	    Application:     "unkey-api",
//	})
//
//	if err != nil {
//	    log.Fatalf("Failed to initialize telemetry: %v", err)
//	}
//	r.RegisterShutdown(shutdown)
//
//	// The runner will call shutdown during graceful termination
func InitGrafana(ctx context.Context, config Config) (func(ctx context.Context) error, error) {
	// Create a resource with common attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNamespace("unkey"),
			semconv.ServiceName(config.Application),
			semconv.ServiceVersion(buildinfo.Version),
			semconv.ServiceInstanceID(config.InstanceID),
			semconv.CloudRegion(config.CloudRegion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Configure OTLP log handler
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	var processor log.Processor = log.NewBatchProcessor(logExporter, log.WithExportBufferSize(512), log.WithExportInterval(15*time.Second))
	processor = minsev.NewLogProcessor(processor, minsev.SeverityInfo)

	//	if config.LogDebug {
	//		processor = minsev.NewLogProcessor(processor, minsev.SeverityDebug)
	//	}

	logProvider := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(processor),
	)

	logger.AddHandler(otelslog.NewHandler(
		config.Application,
		otelslog.WithLoggerProvider(logProvider),
		otelslog.WithVersion(buildinfo.Version),
		otelslog.WithSource(true),
	))

	// Initialize trace exporter with configuration matching the old implementation
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),

	//	otlptracehttp.WithInsecure(), // For local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register shutdown function for trace exporter

	var sampler trace.Sampler

	// Configure the sampler
	if config.TraceSampleRate >= 1.0 {
		sampler = trace.AlwaysSample()
	} else if config.TraceSampleRate <= 0.0 {
		sampler = trace.NeverSample()
	} else {
		sampler = trace.ParentBased(
			trace.TraceIDRatioBased(config.TraceSampleRate),
			trace.WithRemoteParentSampled(trace.AlwaysSample()),
			trace.WithRemoteParentNotSampled(trace.TraceIDRatioBased(config.TraceSampleRate)),
			trace.WithLocalParentSampled(trace.AlwaysSample()),
			trace.WithLocalParentNotSampled(trace.TraceIDRatioBased(config.TraceSampleRate)),
		)
	}

	// Create and register trace provider with the same batch settings as the old code
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(
			traceExporter,
			trace.WithMaxExportBatchSize(512),
			trace.WithBatchTimeout(15*time.Second),
		),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	// Set the global trace provider
	otel.SetTracerProvider(traceProvider)
	tracing.SetGlobalTraceProvider(traceProvider)

	// Shutdown functions run in order during graceful termination. Traces and
	// logs always ship over OTLP; metrics only when a gatherer is provided.
	shutdownFns := []func(context.Context) error{
		traceProvider.Shutdown,
		traceExporter.Shutdown,
		logProvider.Shutdown,
		processor.Shutdown,
		logExporter.Shutdown,
	}

	// Optionally push metrics over OTLP by bridging the provided Prometheus
	// registry. Services that are scraped via /metrics leave the gatherer nil
	// and skip this entirely to avoid double-shipping the same series.
	if config.PrometheusGatherer != nil {
		metricExporter, metricErr := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		)
		if metricErr != nil {
			return nil, fmt.Errorf("failed to create metric exporter: %w", metricErr)
		}

		bridge := prometheus.NewMetricProducer(prometheus.WithGatherer(config.PrometheusGatherer))
		reader := metricsdk.NewPeriodicReader(metricExporter, metricsdk.WithProducer(bridge), metricsdk.WithInterval(60*time.Second))
		meterProvider := metricsdk.NewMeterProvider(metricsdk.WithReader(reader), metricsdk.WithResource(res))
		otel.SetMeterProvider(meterProvider)

		// meterProvider.Shutdown cascades to the reader, which flushes and
		// shuts down the exporter — calling those explicitly afterwards would
		// just return "reader is shutdown" errors.
		shutdownFns = append(shutdownFns, meterProvider.Shutdown)
	}

	// return combined shutdown function that will be called during application termination to cleanly shut down all telemetry components
	return func(ctx context.Context) error {
		for _, fn := range shutdownFns {
			if err := fn(ctx); err != nil {
				return err
			}

		}
		return nil

	}, nil
}
