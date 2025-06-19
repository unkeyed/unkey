package observability

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// BillingMetrics tracks billing-related operations
type BillingMetrics struct {
	logger                 *slog.Logger
	meter                  metric.Meter
	highCardinalityEnabled bool

	// Billing batch metrics
	billingBatchesSent       metric.Int64Counter
	billingBatchSendDuration metric.Float64Histogram
	heartbeatsSent           metric.Int64Counter

	// Metrics collection
	metricsCollected          metric.Int64Counter
	metricsCollectionDuration metric.Float64Histogram
	vmMetricsRequests         metric.Int64Counter
}

// NewBillingMetrics creates new billing metrics
func NewBillingMetrics(logger *slog.Logger, highCardinalityEnabled bool) (*BillingMetrics, error) {
	meter := otel.Meter("unkey.metald.billing")

	bm := &BillingMetrics{ //nolint:exhaustruct // Metric fields are initialized below with error handling
		logger:                 logger.With("component", "billing_metrics"),
		meter:                  meter,
		highCardinalityEnabled: highCardinalityEnabled,
	}

	var err error

	// Billing batch metrics
	if bm.billingBatchesSent, err = meter.Int64Counter(
		"metald_billing_batches_sent_total",
		metric.WithDescription("Total number of billing batches sent"),
	); err != nil {
		return nil, err
	}

	if bm.billingBatchSendDuration, err = meter.Float64Histogram(
		"metald_billing_batch_send_duration_seconds",
		metric.WithDescription("Duration of billing batch send operations"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if bm.heartbeatsSent, err = meter.Int64Counter(
		"metald_heartbeat_sent_total",
		metric.WithDescription("Total number of heartbeats sent to billing service"),
	); err != nil {
		return nil, err
	}

	// Metrics collection
	if bm.metricsCollected, err = meter.Int64Counter(
		"metald_metrics_collected_total",
		metric.WithDescription("Total number of VM metrics collected"),
	); err != nil {
		return nil, err
	}

	if bm.metricsCollectionDuration, err = meter.Float64Histogram(
		"metald_metrics_collection_duration_seconds",
		metric.WithDescription("Duration of metrics collection operations"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, err
	}

	if bm.vmMetricsRequests, err = meter.Int64Counter(
		"metald_vm_metrics_requests_total",
		metric.WithDescription("Total number of VM metrics requests"),
	); err != nil {
		return nil, err
	}

	logger.Info("billing metrics initialized")
	return bm, nil
}

// RecordBillingBatchSent records a billing batch being sent
func (bm *BillingMetrics) RecordBillingBatchSent(ctx context.Context, vmID, customerID string, batchSize int, duration time.Duration) {
	var attrs []attribute.KeyValue
	if bm.highCardinalityEnabled {
		attrs = []attribute.KeyValue{
			attribute.String("vm_id", vmID),
			attribute.String("customer_id", customerID),
		}
	}

	bm.billingBatchesSent.Add(ctx, 1, metric.WithAttributes(attrs...))
	bm.billingBatchSendDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordHeartbeatSent records a heartbeat being sent
func (bm *BillingMetrics) RecordHeartbeatSent(ctx context.Context, instanceID string) {
	bm.heartbeatsSent.Add(ctx, 1, metric.WithAttributes(
		attribute.String("instance_id", instanceID),
	))
}

// RecordMetricsCollected records VM metrics being collected
func (bm *BillingMetrics) RecordMetricsCollected(ctx context.Context, vmID string, metricsCount int, duration time.Duration) {
	var attrs []attribute.KeyValue
	if bm.highCardinalityEnabled {
		attrs = []attribute.KeyValue{
			attribute.String("vm_id", vmID),
		}
	}

	bm.metricsCollected.Add(ctx, int64(metricsCount), metric.WithAttributes(attrs...))
	bm.metricsCollectionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordVMMetricsRequest records a VM metrics request
func (bm *BillingMetrics) RecordVMMetricsRequest(ctx context.Context, vmID string) {
	var attrs []attribute.KeyValue
	if bm.highCardinalityEnabled {
		attrs = []attribute.KeyValue{
			attribute.String("vm_id", vmID),
		}
	}
	bm.vmMetricsRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
}
