package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/version"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/bridges/prometheus"
	"go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
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

	// Version is the current version of your application, allowing you to correlate
	// behavior changes with specific releases.
	Version string

	// TraceSampleRate controls what percentage of traces are sampled.
	// Values range from 0.0 to 1.0, where:
	// - 1.0 means all traces are sampled (100%)
	// - 0.25 means 25% of traces are sampled (the default if not specified)
	// - 0.0 means no traces are sampled (0%)
	//
	// As long as the sampling rate is greater than 0.0, all errors will be sampled.
	TraceSampleRate float64
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
//	    GrafanaEndpoint: "https://otlp-sentinel-prod-us-east-0.grafana.net/otlp",
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
func InitGrafana(ctx context.Context, config Config) (func(ctx context.Context) error, error) {
	// Create a resource with common attributes
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNamespace("unkey"),
			semconv.ServiceName(config.Application),
			semconv.ServiceVersion(config.Version),
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

	logging.AddHandler(otelslog.NewHandler(
		config.Application,
		otelslog.WithLoggerProvider(logProvider),
		otelslog.WithVersion(version.Version),
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

	// Initialize metrics exporter with configuration matching the old implementation
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
	//	otlpmetrichttp.WithInsecure(), // For local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	bridge := prometheus.NewMetricProducer()

	reader := metricsdk.NewPeriodicReader(metricExporter, metricsdk.WithProducer(bridge), metricsdk.WithInterval(60*time.Second))

	// Create and register the metric provider globally
	meterProvider := metricsdk.NewMeterProvider(metricsdk.WithReader(reader), metricsdk.WithResource(res))
	otel.SetMeterProvider(meterProvider)

	err = registerSystemMetrics(meterProvider.Meter(config.Application))
	if err != nil {
		return nil, err
	}

	// return combined shutdown function that will be called during application termination to cleanly shut down all telemetry components
	return func(ctx context.Context) error {
		for _, fn := range []func(context.Context) error{
			meterProvider.Shutdown,
			reader.Shutdown,
			metricExporter.Shutdown,
			traceProvider.Shutdown,
			traceExporter.Shutdown,
			logProvider.Shutdown,
			processor.Shutdown,
			logExporter.Shutdown,
		} {
			if err := fn(ctx); err != nil {
				return err
			}

		}
		return nil

	}, nil
}

func registerSystemMetrics(m metric.Meter) error {

	_, err := m.Float64ObservableGauge("resources_cpu_percent", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {
		cpuPcts, cpuErr := cpu.PercentWithContext(ctx, time.Second, false)
		if cpuErr != nil {
			return cpuErr
		}
		if len(cpuPcts) == 0 {
			return fmt.Errorf("no cpu data")
		}
		o.Observe(cpuPcts[0])

		return nil
	}))
	if err != nil {
		return err
	}

	_, err = m.Float64ObservableGauge("resources_memory_percent", metric.WithFloat64Callback(func(ctx context.Context, o metric.Float64Observer) error {

		vm, vmErr := mem.VirtualMemoryWithContext(ctx)
		if vmErr != nil {
			return vmErr
		}

		o.Observe(vm.UsedPercent)

		return nil
	}))
	if err != nil {
		return err
	}
	return nil
}
