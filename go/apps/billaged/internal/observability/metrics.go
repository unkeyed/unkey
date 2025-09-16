package observability

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// BillingMetrics holds billing-specific metrics
type BillingMetrics struct {
	usageRecordsProcessed  metric.Int64Counter
	aggregationDuration    metric.Float64Histogram
	activeVMs              metric.Int64UpDownCounter
	billingErrors          metric.Int64Counter
	highCardinalityEnabled bool
}

// NewBillingMetrics creates new billing metrics
func NewBillingMetrics(logger *slog.Logger, highCardinalityEnabled bool) (*BillingMetrics, error) {
	meter := meter()
	if meter == nil {
		return nil, fmt.Errorf("OpenTelemetry meter not available")
	}

	usageRecordsProcessed, err := meter.Int64Counter(
		"billaged_usage_records_processed_total",
		metric.WithDescription("Total number of usage records processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create usage records counter: %w", err)
	}

	aggregationDuration, err := meter.Float64Histogram(
		"billaged_aggregation_duration_seconds",
		metric.WithDescription("Time spent aggregating usage metrics"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create aggregation duration histogram: %w", err)
	}

	activeVMs, err := meter.Int64UpDownCounter(
		"billaged_active_vms",
		metric.WithDescription("Number of active VMs being tracked"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active VMs counter: %w", err)
	}

	billingErrors, err := meter.Int64Counter(
		"billaged_billing_errors_total",
		metric.WithDescription("Total number of billing processing errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create billing errors counter: %w", err)
	}

	logger.Info("billing metrics initialized")

	return &BillingMetrics{
		usageRecordsProcessed:  usageRecordsProcessed,
		aggregationDuration:    aggregationDuration,
		activeVMs:              activeVMs,
		billingErrors:          billingErrors,
		highCardinalityEnabled: highCardinalityEnabled,
	}, nil
}

// meter returns the global meter
func meter() metric.Meter {
	return otel.Meter("billaged/billing")
}

// RecordUsageProcessed records that a usage record was processed
func (m *BillingMetrics) RecordUsageProcessed(ctx context.Context, vmID, customerID string) {
	if m != nil {
		var attrs []attribute.KeyValue
		if m.highCardinalityEnabled {
			attrs = []attribute.KeyValue{
				attribute.String("vm_id", vmID),
				attribute.String("customer_id", customerID),
			}
		}
		m.usageRecordsProcessed.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordAggregationDuration records how long aggregation took
func (m *BillingMetrics) RecordAggregationDuration(ctx context.Context, duration float64) {
	if m != nil {
		m.aggregationDuration.Record(ctx, duration)
	}
}

// UpdateActiveVMs updates the number of active VMs
func (m *BillingMetrics) UpdateActiveVMs(ctx context.Context, count int64) {
	if m != nil {
		m.activeVMs.Add(ctx, count)
	}
}

// RecordBillingError records a billing processing error
func (m *BillingMetrics) RecordBillingError(ctx context.Context, errorType string) {
	if m != nil {
		m.billingErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("error_type", errorType),
			),
		)
	}
}
