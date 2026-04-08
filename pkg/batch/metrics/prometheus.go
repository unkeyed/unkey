/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains batch-related metrics for tracking batch processing performance.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Standard histogram buckets for batch size metrics
var batchSizeBuckets = []float64{
	1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000,
}

// Metrics holds all Prometheus collectors for the batch package.
type Metrics struct {
	// BatchSizeDistribution tracks the size of batches when they're flushed as a histogram,
	// labeled by batch name and trigger type.
	// This helps understand batching efficiency and whether batches are being flushed
	// due to size limits or time intervals.
	//
	// Possible trigger values are:
	// - "size_limit": The batch was flushed because it reached the maximum size.
	// - "time_interval": The batch was flushed because the flush interval elapsed.
	// - "close": The batch was flushed because the processor was closed.
	//
	// Example usage:
	//   m.BatchSizeDistribution.WithLabelValues("database_writes", "size_limit").Observe(float64(len(batch)))
	BatchSizeDistribution *prometheus.HistogramVec

	// BatchOperationsTotal tracks the total number of batch flush operations,
	// labeled by batch name, trigger type, and success status.
	// Use this counter to monitor batch processing patterns and error rates.
	//
	// Possible trigger values are:
	// - "size_limit": The batch was flushed because it reached the maximum size.
	// - "time_interval": The batch was flushed because the flush interval elapsed.
	// - "close": The batch was flushed because the processor was closed.
	//
	// Possible status values are:
	// - "success": The batch flush operation completed successfully.
	// - "error": The batch flush operation encountered an error.
	//
	// Example usage:
	//   m.BatchOperationsTotal.WithLabelValues("database_writes", "size_limit", "success").Inc()
	//   m.BatchOperationsTotal.WithLabelValues("log_entries", "time_interval", "error").Inc()
	BatchOperationsTotal *prometheus.CounterVec

	// BatchItemsProcessedTotal tracks the total number of items processed through all batches,
	// labeled by batch name.
	// This is the core throughput metric for understanding how many individual items
	// are being processed by the batch system.
	//
	// Example usage:
	//   m.BatchItemsProcessedTotal.WithLabelValues("database_writes").Add(float64(len(batch)))
	BatchItemsProcessedTotal *prometheus.CounterVec

	// BatchItemsProcessedErrorsTotal tracks the total number of items that resulted in errors
	// during batch processing, labeled by batch name.
	// Use this counter to monitor error rates in batch processing and identify problematic batches.
	//
	// Example usage:
	//   m.BatchItemsProcessedErrorsTotal.WithLabelValues("database_writes").Add(float64(errorCount))
	BatchItemsProcessedErrorsTotal *prometheus.CounterVec
}

// NoopMetrics returns a Metrics instance registered to a discarded registry.
// Metrics are recorded but not exported, making this safe for tests and optional instrumentation.
func NoopMetrics() *Metrics {
	return NewMetrics(prometheus.NewRegistry())
}

// NewMetrics registers and returns a new Metrics instance with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)

	return &Metrics{
		BatchSizeDistribution: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "batch",
				Name:      "size_distribution",
				Help:      "Distribution of batch sizes when flushed",
				Buckets:   batchSizeBuckets,
			},
			[]string{"name", "trigger"},
		),
		BatchOperationsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "batch",
				Name:      "operations_total",
				Help:      "Total number of batch flush operations processed",
			},
			[]string{"name", "trigger", "status"},
		),
		BatchItemsProcessedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "batch",
				Name:      "items_processed_total",
				Help:      "Total number of items processed through batches",
			},
			[]string{"name"},
		),
		BatchItemsProcessedErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "batch",
				Name:      "items_processed_errors_total",
				Help:      "Total number of items processed through batches that resulted in an error",
			},
			[]string{"name"},
		),
	}
}
