package metrics

import "go.opentelemetry.io/otel/metric"

// int64ObservableGauge implements the Int64Observable interface for gauge metrics.
// It provides a way to observe values that can go up and down over time.
type int64ObservableGauge struct {
	m    metric.Meter
	name string
	opts []metric.Int64ObservableGaugeOption
}

// Ensure int64ObservableGauge implements Int64Observable
var _ Int64Observable = (*int64ObservableGauge)(nil)

// RegisterCallback registers a callback function for the int64ObservableGauge.
// This callback will be invoked when the metrics system collects values.
//
// Parameters:
//   - callback: The function that will be called to observe the metric value
//
// Returns:
//   - error: Any error encountered during callback registration
//
// The callback will be invoked periodically by the metrics collection system,
// and should efficiently compute and report the current value of the metric.
//
// Thread-safety: This method is not safe for concurrent use with other methods
// on the same gauge instance. It should typically be called during initialization.
func (g *int64ObservableGauge) RegisterCallback(callback metric.Int64Callback) error {
	_, err := g.m.Int64ObservableGauge(g.name, append(g.opts, metric.WithInt64Callback(callback))...)

	return err
}

// int64ObservableCounter implements the Int64Observable interface for counter metrics.
// It provides a way to observe monotonically increasing values over time.
type int64ObservableCounter struct {
	m    metric.Meter
	name string
	opts []metric.Int64ObservableCounterOption
}

// Ensure int64ObservableCounter implements Int64Observable
var _ Int64Observable = (*int64ObservableCounter)(nil)

// RegisterCallback registers a callback function for the int64ObservableCounter.
// This callback will be invoked when the metrics system collects values.
//
// Parameters:
//   - callback: The function that will be called to observe the metric value
//
// Returns:
//   - error: Any error encountered during callback registration
//
// The callback will be invoked periodically by the metrics collection system,
// and should efficiently compute and report the current value of the metric.
// For counters, the reported value should never decrease.
//
// Thread-safety: This method is not safe for concurrent use with other methods
// on the same counter instance. It should typically be called during initialization.
func (g *int64ObservableCounter) RegisterCallback(callback metric.Int64Callback) error {
	_, err := g.m.Int64ObservableCounter(g.name, append(g.opts, metric.WithInt64Callback(callback))...)

	return err
}
