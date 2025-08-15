package billing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors"
	billingv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/billaged/v1"
	"github.com/unkeyed/unkey/go/gen/proto/deploy/billaged/v1/billagedv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// BillingClient defines the interface for communicating with billaged service
type BillingClient interface {
	// SendMetricsBatch sends a batch of metrics to billaged
	SendMetricsBatch(ctx context.Context, vmID, customerID string, metrics []*types.VMMetrics) error

	// SendHeartbeat sends a heartbeat with active VM list
	SendHeartbeat(ctx context.Context, instanceID string, activeVMs []string) error

	// NotifyVmStarted notifies billaged that a VM has started
	NotifyVmStarted(ctx context.Context, vmID, customerID string, startTime int64) error

	// NotifyVmStopped notifies billaged that a VM has stopped
	NotifyVmStopped(ctx context.Context, vmID string, stopTime int64) error

	// NotifyPossibleGap notifies billaged of a potential data gap
	NotifyPossibleGap(ctx context.Context, vmID string, lastSent, resumeTime int64) error
}

// MockBillingClient provides a mock implementation for development and testing
type MockBillingClient struct {
	logger *slog.Logger
}

// NewMockBillingClient creates a new mock billing client
func NewMockBillingClient(logger *slog.Logger) *MockBillingClient {
	return &MockBillingClient{
		logger: logger.With("component", "mock_billing_client"),
	}
}

func (m *MockBillingClient) SendMetricsBatch(ctx context.Context, vmID, customerID string, metrics []*types.VMMetrics) error {
	m.logger.InfoContext(ctx, "MOCK: sending metrics batch",
		"vm_id", vmID,
		"customer_id", customerID,
		"metrics_count", len(metrics),
	)

	if len(metrics) > 0 {
		first := metrics[0]
		last := metrics[len(metrics)-1]
		m.logger.DebugContext(ctx, "MOCK: batch details",
			"first_timestamp", first.Timestamp.Format("15:04:05.000"),
			"last_timestamp", last.Timestamp.Format("15:04:05.000"),
			"first_cpu_nanos", first.CpuTimeNanos,
			"last_cpu_nanos", last.CpuTimeNanos,
		)
	}

	return nil
}

func (m *MockBillingClient) SendHeartbeat(ctx context.Context, instanceID string, activeVMs []string) error {
	m.logger.DebugContext(ctx, "MOCK: sending heartbeat",
		"instance_id", instanceID,
		"active_vms_count", len(activeVMs),
		"active_vms", activeVMs,
	)
	return nil
}

func (m *MockBillingClient) NotifyVmStarted(ctx context.Context, vmID, customerID string, startTime int64) error {
	m.logger.InfoContext(ctx, "MOCK: VM started notification",
		"vm_id", vmID,
		"customer_id", customerID,
		"start_time", startTime,
	)
	return nil
}

func (m *MockBillingClient) NotifyVmStopped(ctx context.Context, vmID string, stopTime int64) error {
	m.logger.InfoContext(ctx, "MOCK: VM stopped notification",
		"vm_id", vmID,
		"stop_time", stopTime,
	)
	return nil
}

func (m *MockBillingClient) NotifyPossibleGap(ctx context.Context, vmID string, lastSent, resumeTime int64) error {
	m.logger.WarnContext(ctx, "MOCK: possible data gap notification",
		"vm_id", vmID,
		"last_sent", lastSent,
		"resume_time", resumeTime,
		"gap_duration_ms", (resumeTime-lastSent)/1_000_000,
	)
	return nil
}

// Ensure MockBillingClient implements BillingClient interface
var _ BillingClient = (*MockBillingClient)(nil)

// ConnectRPCBillingClient implements real ConnectRPC client for billaged
type ConnectRPCBillingClient struct {
	endpoint string
	logger   *slog.Logger
	client   billagedv1connect.BillingServiceClient
}

func NewConnectRPCBillingClient(endpoint string, logger *slog.Logger) *ConnectRPCBillingClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// AIDEV-NOTE: Using debug interceptor for comprehensive error tracking
	billingClient := billagedv1connect.NewBillingServiceClient(
		httpClient,
		endpoint,
		connect.WithInterceptors(
			observability.DebugInterceptor(logger, "billaged"),
		),
	)

	return &ConnectRPCBillingClient{
		endpoint: endpoint,
		logger:   logger.With("component", "connectrpc_billing_client"),
		client:   billingClient,
	}
}

// NewConnectRPCBillingClientWithHTTP creates a billing client with a custom HTTP client (for TLS)
func NewConnectRPCBillingClientWithHTTP(endpoint string, logger *slog.Logger, httpClient *http.Client) *ConnectRPCBillingClient {
	// Use provided HTTP client which may have TLS configuration
	// AIDEV-NOTE: Using shared client interceptors for consistency across services
	clientInterceptors := interceptors.NewDefaultClientInterceptors("metald", logger)
	// Add debug interceptor for detailed error tracking
	clientInterceptors = append(clientInterceptors,
		observability.DebugInterceptor(logger, "billaged"),
	)

	// Convert UnaryInterceptorFunc to Interceptor
	var interceptorList []connect.Interceptor
	for _, interceptor := range clientInterceptors {
		interceptorList = append(interceptorList, connect.Interceptor(interceptor))
	}

	billingClient := billagedv1connect.NewBillingServiceClient(
		httpClient,
		endpoint,
		connect.WithInterceptors(interceptorList...),
	)

	return &ConnectRPCBillingClient{
		endpoint: endpoint,
		logger:   logger.With("component", "connectrpc_billing_client"),
		client:   billingClient,
	}
}

func (c *ConnectRPCBillingClient) SendMetricsBatch(ctx context.Context, vmID, customerID string, metrics []*types.VMMetrics) error {
	// Convert metald VMMetrics to billaged VMMetrics
	billingMetrics := make([]*billingv1.VMMetrics, len(metrics))
	for i, m := range metrics {
		billingMetrics[i] = &billingv1.VMMetrics{
			Timestamp:        timestamppb.New(m.Timestamp),
			CpuTimeNanos:     m.CpuTimeNanos,
			MemoryUsageBytes: m.MemoryUsageBytes,
			DiskReadBytes:    m.DiskReadBytes,
			DiskWriteBytes:   m.DiskWriteBytes,
			NetworkRxBytes:   m.NetworkRxBytes,
			NetworkTxBytes:   m.NetworkTxBytes,
		}
	}

	req := &billingv1.SendMetricsBatchRequest{
		VmId:       vmID,
		CustomerId: customerID,
		Metrics:    billingMetrics,
	}

	resp, err := c.client.SendMetricsBatch(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.ErrorContext(ctx, "billaged connection error",
				"error", err.Error(),
				"code", connectErr.Code().String(),
				"message", connectErr.Message(),
				"vm_id", vmID,
				"customer_id", customerID,
				"metrics_count", len(metrics),
				"operation", "SendMetricsBatch",
			)
		} else {
			c.logger.ErrorContext(ctx, "failed to send metrics batch",
				"error", err.Error(),
				"vm_id", vmID,
				"customer_id", customerID,
				"metrics_count", len(metrics),
				"operation", "SendMetricsBatch",
			)
		}
		return fmt.Errorf("failed to send metrics batch: %w", err)
	}

	if !resp.Msg.GetSuccess() {
		return fmt.Errorf("billaged rejected metrics batch: %s", resp.Msg.GetMessage())
	}

	c.logger.DebugContext(ctx, "sent metrics batch to billaged",
		"vm_id", vmID,
		"customer_id", customerID,
		"metrics_count", len(metrics),
		"message", resp.Msg.GetMessage(),
	)

	return nil
}

func (c *ConnectRPCBillingClient) SendHeartbeat(ctx context.Context, instanceID string, activeVMs []string) error {
	req := &billingv1.SendHeartbeatRequest{
		InstanceId: instanceID,
		ActiveVms:  activeVMs,
	}

	resp, err := c.client.SendHeartbeat(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.ErrorContext(ctx, "billaged connection error",
				"error", err.Error(),
				"code", connectErr.Code().String(),
				"message", connectErr.Message(),
				"instance_id", instanceID,
				"active_vms_count", len(activeVMs),
				"operation", "SendHeartbeat",
			)
		} else {
			c.logger.ErrorContext(ctx, "failed to send heartbeat",
				"error", err.Error(),
				"instance_id", instanceID,
				"active_vms_count", len(activeVMs),
				"operation", "SendHeartbeat",
			)
		}
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	if !resp.Msg.GetSuccess() {
		return fmt.Errorf("billaged rejected heartbeat")
	}

	return nil
}

func (c *ConnectRPCBillingClient) NotifyVmStarted(ctx context.Context, vmID, customerID string, startTime int64) error {
	req := &billingv1.NotifyVmStartedRequest{
		VmId:       vmID,
		CustomerId: customerID,
		StartTime:  startTime,
	}

	resp, err := c.client.NotifyVmStarted(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.ErrorContext(ctx, "billaged connection error",
				"error", err.Error(),
				"code", connectErr.Code().String(),
				"message", connectErr.Message(),
				"vm_id", vmID,
				"customer_id", customerID,
				"start_time", startTime,
				"operation", "NotifyVmStarted",
			)
		} else {
			c.logger.ErrorContext(ctx, "failed to notify VM started",
				"error", err.Error(),
				"vm_id", vmID,
				"customer_id", customerID,
				"start_time", startTime,
				"operation", "NotifyVmStarted",
			)
		}
		return fmt.Errorf("failed to notify VM started: %w", err)
	}

	if !resp.Msg.GetSuccess() {
		return fmt.Errorf("billaged rejected VM started notification")
	}

	return nil
}

func (c *ConnectRPCBillingClient) NotifyVmStopped(ctx context.Context, vmID string, stopTime int64) error {
	req := &billingv1.NotifyVmStoppedRequest{
		VmId:     vmID,
		StopTime: stopTime,
	}

	resp, err := c.client.NotifyVmStopped(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.ErrorContext(ctx, "billaged connection error",
				"error", err.Error(),
				"code", connectErr.Code().String(),
				"message", connectErr.Message(),
				"vm_id", vmID,
				"stop_time", stopTime,
				"operation", "NotifyVmStopped",
			)
		} else {
			c.logger.ErrorContext(ctx, "failed to notify VM stopped",
				"error", err.Error(),
				"vm_id", vmID,
				"stop_time", stopTime,
				"operation", "NotifyVmStopped",
			)
		}
		return fmt.Errorf("failed to notify VM stopped: %w", err)
	}

	if !resp.Msg.GetSuccess() {
		return fmt.Errorf("billaged rejected VM stopped notification")
	}

	return nil
}

func (c *ConnectRPCBillingClient) NotifyPossibleGap(ctx context.Context, vmID string, lastSent, resumeTime int64) error {
	req := &billingv1.NotifyPossibleGapRequest{
		VmId:       vmID,
		LastSent:   lastSent,
		ResumeTime: resumeTime,
	}

	resp, err := c.client.NotifyPossibleGap(ctx, connect.NewRequest(req))
	if err != nil {
		// AIDEV-NOTE: Enhanced debug logging for connection errors
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			c.logger.ErrorContext(ctx, "billaged connection error",
				"error", err.Error(),
				"code", connectErr.Code().String(),
				"message", connectErr.Message(),
				"vm_id", vmID,
				"last_sent", lastSent,
				"resume_time", resumeTime,
				"gap_duration_ms", (resumeTime-lastSent)/1_000_000,
				"operation", "NotifyPossibleGap",
			)
		} else {
			c.logger.ErrorContext(ctx, "failed to notify possible gap",
				"error", err.Error(),
				"vm_id", vmID,
				"last_sent", lastSent,
				"resume_time", resumeTime,
				"gap_duration_ms", (resumeTime-lastSent)/1_000_000,
				"operation", "NotifyPossibleGap",
			)
		}
		return fmt.Errorf("failed to notify possible gap: %w", err)
	}

	if !resp.Msg.GetSuccess() {
		return fmt.Errorf("billaged rejected possible gap notification")
	}

	return nil
}

// Ensure ConnectRPCBillingClient implements BillingClient interface
var _ BillingClient = (*ConnectRPCBillingClient)(nil)
