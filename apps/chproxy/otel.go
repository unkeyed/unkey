package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type TelemetryConfig struct {
	LogHandler        slog.Handler
	LogHTTPOptions    []otlploghttp.Option
	Meter             metric.Meter
	MetricHTTPOptions []otlpmetrichttp.Option
	Metrics           struct {
		BatchCounter  metric.Int64Counter
		ErrorCounter  metric.Int64Counter
		FlushCounter  metric.Int64Counter
		FlushDuration metric.Float64Histogram
		RowCounter    metric.Int64Counter
	}
	TraceHTTPOptions []otlptracehttp.Option
	Tracer           trace.Tracer
}

var currentBufferSize int64

// SetBufferSize updates the current buffer size for metrics
func SetBufferSize(size int64) {
	currentBufferSize = size
}

// GetBufferSize returns the current buffer size
func GetBufferSize() int64 {
	return currentBufferSize
}

// SetupTelemetry initializes OTEL tracing, metrics, and logging
func setupTelemetry(ctx context.Context, config *Config) (*TelemetryConfig, func(context.Context) error, error) {
	telemetryConfig := &TelemetryConfig{}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTEL resource: %w", err)
	}

	// Configure metric/meter
	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create OTEL metrics exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExporter,
				sdkmetric.WithInterval(10*time.Second),
			),
		),
	)
	otel.SetMeterProvider(meterProvider)
	telemetryConfig.Meter = meterProvider.Meter(config.ServiceName)

	logExporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(logExporter),
		),
	)

	otlpHandler := otelslog.NewHandler(
		config.ServiceName,
		otelslog.WithLoggerProvider(logProvider),
	)

	telemetryConfig.LogHandler = otlpHandler

	// Configure tracer
	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(traceProvider)
	telemetryConfig.Tracer = traceProvider.Tracer(config.ServiceName)

	// Initialize metrics
	var err1, err2, err3, err4, err5 error
	telemetryConfig.Metrics.BatchCounter, err1 = telemetryConfig.Meter.Int64Counter(
		"clickhouse_batches_total",
		metric.WithDescription("Total number of batches sent to Clickhouse"),
	)
	telemetryConfig.Metrics.RowCounter, err2 = telemetryConfig.Meter.Int64Counter(
		"clickhouse_rows_total",
		metric.WithDescription("Total number of rows sent to Clickhouse"),
	)
	telemetryConfig.Metrics.FlushCounter, err3 = telemetryConfig.Meter.Int64Counter(
		"clickhouse_flushes_total",
		metric.WithDescription("Total number of flush operations"),
		metric.WithUnit("{flush}"),
	)
	telemetryConfig.Metrics.ErrorCounter, err4 = telemetryConfig.Meter.Int64Counter(
		"clickhouse_errors_total",
		metric.WithDescription("Total number of errors"),
	)
	telemetryConfig.Metrics.FlushDuration, err5 = telemetryConfig.Meter.Float64Histogram(
		"clickhouse_flush_duration_seconds",
		metric.WithDescription("Duration of flush operations"),
		metric.WithUnit("s"),
	)

	for _, err := range []error{err1, err2, err3, err4, err5} {
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create metric: %w", err)
		}
	}

	_, err = telemetryConfig.Meter.Int64ObservableGauge(
		"clickhouse_buffer_size",
		metric.WithDescription("Current number of rows in buffer"),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(GetBufferSize())
			return nil
		}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create buffer size gauge: %w", err)
	}

	// Return a cleanup function (should be nil but might be an actual error)
	cleanup := func(ctx context.Context) error {
		err := meterProvider.Shutdown(ctx)
		if err != nil {
			return err
		}

		if err := traceProvider.Shutdown(ctx); err != nil {
			return err
		}

		// Actually shutdown the log exporter
		return logExporter.Shutdown(ctx)
	}

	return telemetryConfig, cleanup, nil
}
