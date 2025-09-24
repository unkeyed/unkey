package client

import (
	billingv1 "github.com/unkeyed/unkey/go/gen/proto/billaged/v1"
)

// AIDEV-NOTE: Type definitions for billaged client requests and responses
// These provide a clean interface that wraps the protobuf types

// SendMetricsBatchRequest represents a request to send a batch of VM metrics
type SendMetricsBatchRequest struct {
	VmID       string
	CustomerID string
	Metrics    []*billingv1.VMMetrics
}

// SendMetricsBatchResponse represents the response from sending metrics batch
type SendMetricsBatchResponse struct {
	Success bool
	Message string
}

// SendHeartbeatRequest represents a request to send a heartbeat
type SendHeartbeatRequest struct {
	InstanceID string
	ActiveVMs  []string
}

// SendHeartbeatResponse represents the response from sending heartbeat
type SendHeartbeatResponse struct {
	Success bool
}

// NotifyVmStartedRequest represents a request to notify that a VM has started
type NotifyVmStartedRequest struct {
	VmID       string
	CustomerID string
	StartTime  int64
}

// NotifyVmStartedResponse represents the response from notifying VM started
type NotifyVmStartedResponse struct {
	Success bool
}

// NotifyVmStoppedRequest represents a request to notify that a VM has stopped
type NotifyVmStoppedRequest struct {
	VmID     string
	StopTime int64
}

// NotifyVmStoppedResponse represents the response from notifying VM stopped
type NotifyVmStoppedResponse struct {
	Success bool
}

// NotifyPossibleGapRequest represents a request to notify about a possible gap in metrics
type NotifyPossibleGapRequest struct {
	VmID       string
	LastSent   int64
	ResumeTime int64
}

// NotifyPossibleGapResponse represents the response from notifying possible gap
type NotifyPossibleGapResponse struct {
	Success bool
}
