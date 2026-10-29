/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains batch-related metrics for tracking batch processing performance.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

// Standard histogram buckets for batch size metrics
var batchSizeBuckets = []float64{
	1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000,
}

var (
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
	//   metrics.BatchSizeDistribution.WithLabelValues("database_writes", "size_limit").Observe(float64(len(batch)))
	BatchSizeDistribution = vec.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "batch",
			Name:      "size_distribution",
			Help:      "Distribution of batch sizes when flushed",
			Buckets:   batchSizeBuckets,
		},
		vec.Dynamic("name"),
		vec.Values("trigger", "size_limit", "time_interval", "close"),
	)

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
	//   metrics.BatchOperationsTotal.WithLabelValues("database_writes", "size_limit", "success").Inc()
	//   metrics.BatchOperationsTotal.WithLabelValues("log_entries", "time_interval", "error").Inc()
	BatchOperationsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "batch",
			Name:      "operations_total",
			Help:      "Total number of batch flush operations processed",
		},
		vec.Dynamic("name"),
		vec.Values("trigger", "size_limit", "time_interval", "close"),
		vec.Values("status", "success", "error"),
	)

	// BatchItemsProcessedTotal tracks the total number of items processed through all batches,
	// labeled by batch name.
	// This is the core throughput metric for understanding how many individual items
	// are being processed by the batch system.
	//
	// Example usage:
	//   metrics.BatchItemsProcessedTotal.WithLabelValues("database_writes").Add(float64(len(batch)))
	BatchItemsProcessedTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "batch",
			Name:      "items_processed_total",
			Help:      "Total number of items processed through batches",
		},
		vec.Dynamic("name"),
	)

	// BatchItemsProcessedErrorsTotal tracks the total number of items that resulted in errors
	// during batch processing, labeled by batch name.
	// Use this counter to monitor error rates in batch processing and identify problematic batches.
	//
	// Example usage:
	//   metrics.BatchItemsProcessedErrorsTotal.WithLabelValues("database_writes").Add(float64(errorCount))
	BatchItemsProcessedErrorsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "batch",
			Name:      "items_processed_errors_total",
			Help:      "Total number of items processed through batches that resulted in an error",
		},
		vec.Dynamic("name"),
	)
)
