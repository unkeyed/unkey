package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/metald/internal/config"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Providers holds the OpenTelemetry providers
type Providers struct {
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	PrometheusHTTP http.Handler
	Shutdown       func(context.Context) error
}

// InitProviders initializes OpenTelemetry providers
func InitProviders(ctx context.Context, cfg *config.Config, version string, logger *slog.Logger) (*Providers, error) {
	if !cfg.OpenTelemetry.Enabled {
		// Return no-op providers
		return &Providers{
			TracerProvider: noop.NewTracerProvider(),
			MeterProvider:  nil,
			PrometheusHTTP: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("OpenTelemetry is disabled"))
			}),
			Shutdown: func(context.Context) error { return nil },
		}, nil
	}

	// AIDEV-NOTE: Schema conflict fix - Using semconv v1.24.0 with OTEL v1.36.0
	// and resource.New() without auto-detection resolves conflicting Schema URLs
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNamespace("unkey"),
			semconv.ServiceName(cfg.OpenTelemetry.ServiceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTEL resource: %w", err)
	}

	// Initialize trace provider
	tracerProvider, tracerShutdown, err := initTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	// Initialize meter provider
	meterProvider, promHandler, meterShutdown, err := initMeterProvider(ctx, cfg, res)
	if err != nil {
		if shutdownErr := tracerShutdown(ctx); shutdownErr != nil {
			logger.ErrorContext(ctx, "Failed to shutdown tracer", "error", shutdownErr)
		}
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Set global providers
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Combined shutdown function
	shutdown := func(ctx context.Context) error {
		var errs []error

		if err := tracerShutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown error: %w", err))
		}

		if err := meterShutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter shutdown error: %w", err))
		}

		if len(errs) > 0 {
			return errors.Join(errs...)
		}

		return nil
	}

	return &Providers{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
		PrometheusHTTP: promHandler,
		Shutdown:       shutdown,
	}, nil
}

// initTracerProvider initializes the tracer provider
func initTracerProvider(ctx context.Context, cfg *config.Config, res *resource.Resource) (trace.TracerProvider, func(context.Context) error, error) {
	// Create OTLP trace exporter
	traceExporter, err := otlptrace.New(ctx,
		otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(cfg.OpenTelemetry.OTLPEndpoint),
			otlptracehttp.WithInsecure(), // For local development
			otlptracehttp.WithTimeout(30*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create sampler with parent-based + ratio
	// Note: Error sampling is handled at the span level when RecordError is called
	ratioSampler := sdktrace.TraceIDRatioBased(cfg.OpenTelemetry.TracingSamplingRate)
	parentBasedSampler := sdktrace.ParentBased(ratioSampler)

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(parentBasedSampler),
	)

	return tp, tp.Shutdown, nil
}

// initMeterProvider initializes the meter provider
func initMeterProvider(ctx context.Context, cfg *config.Config, res *resource.Resource) (metric.MeterProvider, http.Handler, func(context.Context) error, error) {
	var readers []sdkmetric.Reader

	// OTLP metric exporter
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(cfg.OpenTelemetry.OTLPEndpoint),
		otlpmetrichttp.WithInsecure(), // For local development
		otlpmetrichttp.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	readers = append(readers, sdkmetric.NewPeriodicReader(
		metricExporter,
		sdkmetric.WithInterval(15*time.Second),
	))

	// Prometheus exporter
	var promHandler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Prometheus metrics disabled"))
	})

	if cfg.OpenTelemetry.PrometheusEnabled {
		promExporter, err := prometheus.New()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
		}
		readers = append(readers, promExporter)
		promHandler = promhttp.Handler()
	}

	// Create meter provider with readers
	mpOpts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	for _, reader := range readers {
		mpOpts = append(mpOpts, sdkmetric.WithReader(reader))
	}
	mp := sdkmetric.NewMeterProvider(mpOpts...)

	return mp, promHandler, mp.Shutdown, nil
}

// RecordError records an error in the current span and sets the status
func RecordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// HTTPStatusCode returns the appropriate trace status code for an HTTP status
func HTTPStatusCode(httpStatus int) codes.Code {
	if httpStatus >= 200 && httpStatus < 400 {
		return codes.Ok
	}
	return codes.Error
}

// SpanKindFromMethod returns the appropriate span kind for a method
func SpanKindFromMethod(method string) trace.SpanKind {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return trace.SpanKindClient
	default:
		return trace.SpanKindInternal
	}
}

// ServiceAttributes returns common service attributes
func ServiceAttributes(cfg *config.Config, version string) []attribute.KeyValue {
	return []attribute.KeyValue{
		semconv.ServiceName(cfg.OpenTelemetry.ServiceName),
		semconv.ServiceVersion(version),
		semconv.ServiceNamespace("unkey"),
	}
}
