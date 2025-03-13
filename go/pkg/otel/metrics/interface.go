package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// Int64Counter represents a metric that accumulates int64 values.
// It's a wrapper around OpenTelemetry's metric.Int64Counter that simplifies its usage.
type Int64Counter interface {
	// Add increments the counter by the given value.
	//
	// Parameters:
	//   - ctx: The context for the operation, which can carry tracing information
	//   - incr: The amount to increment the counter by
	//   - options: Optional metric.AddOption values like attributes
	//
	// This method is safe for concurrent use.
	Add(ctx context.Context, incr int64, options ...metric.AddOption)
}

// Int64Observable represents a metric that reports values through a callback function.
// This is used for metrics that need to be computed on-demand when they're collected.
type Int64Observable interface {
	// RegisterCallback registers a callback function that will be called when the metric
	// is collected.
	//
	// Parameters:
	//   - callback: The function that will be called to observe the metric value
	//
	// Returns:
	//   - error: Any error encountered during callback registration
	//
	// This method is not guaranteed to be safe for concurrent use and should typically
	// be called during application initialization.
	RegisterCallback(metric.Int64Callback) error
}
