// Package interceptors provides shared ConnectRPC interceptors for observability and tenant management
// across all Unkey services. These interceptors handle metrics collection, distributed tracing,
// structured logging, and tenant authentication in a consistent manner.
package interceptors

import (
	"log/slog"

	"go.opentelemetry.io/otel/metric"
)

// Options holds configuration for interceptors.
type Options struct {
	// ServiceName is the name of the service using the interceptor.
	ServiceName string

	// Logger is the structured logger to use. If nil, logging interceptor is disabled.
	Logger *slog.Logger

	// Meter is the OpenTelemetry meter for metrics. If nil, metrics are disabled.
	Meter metric.Meter

	// EnableActiveRequestsMetric controls whether to track active requests count.
	EnableActiveRequestsMetric bool

	// EnableRequestDurationMetric controls whether to record request duration histogram.
	EnableRequestDurationMetric bool

	// EnablePanicStackTrace controls whether to log full stack traces on panic.
	EnablePanicStackTrace bool

	// EnableErrorResampling controls whether to create additional spans for errors
	// when the main span is not sampled.
	EnableErrorResampling bool
}

// Option is a function that configures Options.
type Option func(*Options)

// WithServiceName sets the service name for interceptors.
func WithServiceName(name string) Option {
	return func(o *Options) {
		o.ServiceName = name
	}
}

// WithLogger sets the logger for the logging interceptor.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithMeter sets the OpenTelemetry meter for metrics collection.
func WithMeter(meter metric.Meter) Option {
	return func(o *Options) {
		o.Meter = meter
	}
}

// WithActiveRequestsMetric enables tracking of active requests count.
func WithActiveRequestsMetric(enabled bool) Option {
	return func(o *Options) {
		o.EnableActiveRequestsMetric = enabled
	}
}

// WithRequestDurationMetric enables request duration histogram.
func WithRequestDurationMetric(enabled bool) Option {
	return func(o *Options) {
		o.EnableRequestDurationMetric = enabled
	}
}

// WithPanicStackTrace enables logging of full stack traces on panic.
func WithPanicStackTrace(enabled bool) Option {
	return func(o *Options) {
		o.EnablePanicStackTrace = enabled
	}
}

// WithErrorResampling enables creation of additional spans for errors
// when the main span is not sampled.
func WithErrorResampling(enabled bool) Option {
	return func(o *Options) {
		o.EnableErrorResampling = enabled
	}
}

// applyOptions creates an Options struct from the provided options.
func applyOptions(opts []Option) *Options {
	options := &Options{
		ServiceName:                 "unknown",
		EnableActiveRequestsMetric:  true,
		EnableRequestDurationMetric: false,
		EnablePanicStackTrace:       true,
		EnableErrorResampling:       true,
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}
