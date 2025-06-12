package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	billingv1 "github.com/unkeyed/unkey/go/deploy/billaged/gen/billing/v1"
	"github.com/unkeyed/unkey/go/deploy/billaged/gen/billing/v1/billingv1connect"
	"github.com/unkeyed/unkey/go/deploy/billaged/internal/aggregator"
	"github.com/unkeyed/unkey/go/deploy/billaged/internal/observability"
)

// BillingService implements the billaged ConnectRPC service
type BillingService struct {
	logger     *slog.Logger
	aggregator *aggregator.Aggregator
	metrics    *observability.BillingMetrics
}

// NewBillingService creates a new billing service
func NewBillingService(logger *slog.Logger, agg *aggregator.Aggregator, metrics *observability.BillingMetrics) *BillingService {
	return &BillingService{
		logger:     logger.With("component", "billing_service"),
		aggregator: agg,
		metrics:    metrics,
	}
}

// SendMetricsBatch processes a batch of VM metrics from metald
func (s *BillingService) SendMetricsBatch(
	ctx context.Context,
	req *connect.Request[billingv1.SendMetricsBatchRequest],
) (*connect.Response[billingv1.SendMetricsBatchResponse], error) {
	vmID := req.Msg.VmId
	customerID := req.Msg.CustomerId
	metrics := req.Msg.Metrics

	s.logger.Info("received metrics batch",
		"vm_id", vmID,
		"customer_id", customerID,
		"metrics_count", len(metrics),
	)

	if len(metrics) == 0 {
		return connect.NewResponse(&billingv1.SendMetricsBatchResponse{
			Success: false,
			Message: "no metrics provided",
		}), nil
	}

	// Log first and last metric for debugging
	first := metrics[0]
	last := metrics[len(metrics)-1]
	s.logger.Debug("metrics batch details",
		"vm_id", vmID,
		"first_timestamp", first.Timestamp.AsTime().Format("15:04:05.000"),
		"last_timestamp", last.Timestamp.AsTime().Format("15:04:05.000"),
		"first_cpu_nanos", first.CpuTimeNanos,
		"last_cpu_nanos", last.CpuTimeNanos,
		"timespan_ms", last.Timestamp.AsTime().Sub(first.Timestamp.AsTime()).Milliseconds(),
	)

	// Record metrics
	start := time.Now()
	if s.metrics != nil {
		s.metrics.RecordUsageProcessed(ctx, vmID, customerID)
	}

	// Process metrics through aggregator
	s.aggregator.ProcessMetricsBatch(vmID, customerID, metrics)

	// Record aggregation duration
	if s.metrics != nil {
		s.metrics.RecordAggregationDuration(ctx, time.Since(start).Seconds())
	}

	return connect.NewResponse(&billingv1.SendMetricsBatchResponse{
		Success: true,
		Message: fmt.Sprintf("processed %d metrics", len(metrics)),
	}), nil
}

// SendHeartbeat processes heartbeat from metald with active VM list
func (s *BillingService) SendHeartbeat(
	ctx context.Context,
	req *connect.Request[billingv1.SendHeartbeatRequest],
) (*connect.Response[billingv1.SendHeartbeatResponse], error) {
	instanceID := req.Msg.InstanceId
	activeVMs := req.Msg.ActiveVms

	s.logger.Debug("received heartbeat",
		"instance_id", instanceID,
		"active_vms_count", len(activeVMs),
		"active_vms", activeVMs,
	)

	// Heartbeat processing could include health checks,
	// gap detection, or VM lifecycle validation here

	return connect.NewResponse(&billingv1.SendHeartbeatResponse{
		Success: true,
	}), nil
}

// NotifyVmStarted handles VM start notifications
func (s *BillingService) NotifyVmStarted(
	ctx context.Context,
	req *connect.Request[billingv1.NotifyVmStartedRequest],
) (*connect.Response[billingv1.NotifyVmStartedResponse], error) {
	vmID := req.Msg.VmId
	customerID := req.Msg.CustomerId
	startTime := req.Msg.StartTime

	s.logger.Info("VM started notification",
		"vm_id", vmID,
		"customer_id", customerID,
		"start_time", startTime,
	)

	s.aggregator.NotifyVMStarted(vmID, customerID, startTime)

	return connect.NewResponse(&billingv1.NotifyVmStartedResponse{
		Success: true,
	}), nil
}

// NotifyVmStopped handles VM stop notifications
func (s *BillingService) NotifyVmStopped(
	ctx context.Context,
	req *connect.Request[billingv1.NotifyVmStoppedRequest],
) (*connect.Response[billingv1.NotifyVmStoppedResponse], error) {
	vmID := req.Msg.VmId
	stopTime := req.Msg.StopTime

	s.logger.Info("VM stopped notification",
		"vm_id", vmID,
		"stop_time", stopTime,
	)

	s.aggregator.NotifyVMStopped(vmID, stopTime)

	return connect.NewResponse(&billingv1.NotifyVmStoppedResponse{
		Success: true,
	}), nil
}

// NotifyPossibleGap handles data gap notifications
func (s *BillingService) NotifyPossibleGap(
	ctx context.Context,
	req *connect.Request[billingv1.NotifyPossibleGapRequest],
) (*connect.Response[billingv1.NotifyPossibleGapResponse], error) {
	vmID := req.Msg.VmId
	lastSent := req.Msg.LastSent
	resumeTime := req.Msg.ResumeTime

	gapDurationMs := (resumeTime - lastSent) / 1_000_000

	s.logger.Warn("possible data gap notification",
		"vm_id", vmID,
		"last_sent", lastSent,
		"resume_time", resumeTime,
		"gap_duration_ms", gapDurationMs,
	)

	// Gap handling could include:
	// - Marking billing periods as incomplete
	// - Triggering reconciliation processes
	// - Alerting operations teams

	return connect.NewResponse(&billingv1.NotifyPossibleGapResponse{
		Success: true,
	}), nil
}

// Ensure BillingService implements the interface
var _ billingv1connect.BillingServiceHandler = (*BillingService)(nil)
