package aggregator

import (
	"context"
	"log/slog"
	"sync"
	"time"

	billingv1 "github.com/unkeyed/unkey/go/gen/proto/billaged/v1"
)

// VMUsageData tracks usage for a single VM
type VMUsageData struct {
	VMID                string
	CustomerID          string
	StartTime           time.Time
	LastUpdate          time.Time
	TotalCPUNanos       int64
	TotalMemoryBytes    int64
	TotalDiskReadBytes  int64
	TotalDiskWriteBytes int64
	TotalNetworkRxBytes int64
	TotalNetworkTxBytes int64
	SampleCount         int64

	// For calculating rates/deltas
	LastCPUNanos       int64
	LastMemoryBytes    int64
	LastDiskReadBytes  int64
	LastDiskWriteBytes int64
	LastNetworkRxBytes int64
	LastNetworkTxBytes int64
}

// UsageSummary contains aggregated usage over a time period
type UsageSummary struct {
	VMID       string
	CustomerID string
	Period     time.Duration
	StartTime  time.Time
	EndTime    time.Time

	// CPU time actually used (not just allocated)
	CPUTimeUsedNanos int64
	CPUTimeUsedMs    int64

	// Memory usage statistics
	AvgMemoryUsageBytes int64
	MaxMemoryUsageBytes int64

	// Disk I/O totals
	DiskReadBytes  int64
	DiskWriteBytes int64
	TotalDiskIO    int64

	// Network I/O totals
	NetworkRxBytes int64
	NetworkTxBytes int64
	TotalNetworkIO int64

	// Overall resource usage score (for billing)
	ResourceScore float64

	SampleCount int64
}

// Aggregator collects and aggregates VM usage data for billing
type Aggregator struct {
	logger    *slog.Logger
	mu        sync.RWMutex
	vmData    map[string]*VMUsageData // vmID -> usage data
	customers map[string][]string     // customerID -> []vmID

	// Aggregation interval (configurable)
	aggregationInterval time.Duration

	// Callbacks for reporting
	onUsageSummary func(*UsageSummary)
}

// NewAggregator creates a new billing aggregator
func NewAggregator(logger *slog.Logger, aggregationInterval time.Duration) *Aggregator {
	return &Aggregator{ //nolint:exhaustruct // mu and onUsageSummary fields use appropriate zero values and are set later
		logger:              logger.With("component", "billing_aggregator"),
		vmData:              make(map[string]*VMUsageData),
		customers:           make(map[string][]string),
		aggregationInterval: aggregationInterval,
	}
}

// SetUsageSummaryCallback sets the callback for when usage summaries are ready
func (a *Aggregator) SetUsageSummaryCallback(callback func(*UsageSummary)) {
	a.onUsageSummary = callback
}

// ProcessMetricsBatch processes a batch of metrics from metald
func (a *Aggregator) ProcessMetricsBatch(vmID, customerID string, metrics []*billingv1.VMMetrics) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(metrics) == 0 {
		return
	}

	// Get or create VM usage data
	vmUsage, exists := a.vmData[vmID]
	if !exists {
		vmUsage = &VMUsageData{ //nolint:exhaustruct // Other usage tracking fields are initialized during metric processing
			VMID:       vmID,
			CustomerID: customerID,
			StartTime:  metrics[0].GetTimestamp().AsTime(),
		}
		a.vmData[vmID] = vmUsage

		// Track customer -> VM mapping
		a.customers[customerID] = append(a.customers[customerID], vmID)
	}

	// Process each metric in the batch
	for _, metric := range metrics {
		a.processMetric(vmUsage, metric)
	}

	vmUsage.LastUpdate = time.Now()
	vmUsage.SampleCount += int64(len(metrics))

	a.logger.Debug("processed metrics batch",
		"vm_id", vmID,
		"customer_id", customerID,
		"metrics_count", len(metrics),
		"total_samples", vmUsage.SampleCount,
	)
}

// processMetric processes a single metric and updates usage data
func (a *Aggregator) processMetric(vmUsage *VMUsageData, metric *billingv1.VMMetrics) {
	// Calculate deltas (incremental usage)
	cpuDelta := metric.GetCpuTimeNanos() - vmUsage.LastCPUNanos
	diskReadDelta := metric.GetDiskReadBytes() - vmUsage.LastDiskReadBytes
	diskWriteDelta := metric.GetDiskWriteBytes() - vmUsage.LastDiskWriteBytes
	netRxDelta := metric.GetNetworkRxBytes() - vmUsage.LastNetworkRxBytes
	netTxDelta := metric.GetNetworkTxBytes() - vmUsage.LastNetworkTxBytes

	// Only add positive deltas (handle counter resets gracefully)
	if cpuDelta > 0 {
		vmUsage.TotalCPUNanos += cpuDelta
	}
	if diskReadDelta > 0 {
		vmUsage.TotalDiskReadBytes += diskReadDelta
	}
	if diskWriteDelta > 0 {
		vmUsage.TotalDiskWriteBytes += diskWriteDelta
	}
	if netRxDelta > 0 {
		vmUsage.TotalNetworkRxBytes += netRxDelta
	}
	if netTxDelta > 0 {
		vmUsage.TotalNetworkTxBytes += netTxDelta
	}

	// Memory is a point-in-time value, track max and average
	if metric.GetMemoryUsageBytes() > vmUsage.TotalMemoryBytes {
		vmUsage.TotalMemoryBytes = metric.GetMemoryUsageBytes()
	}

	// Update last values for next delta calculation
	vmUsage.LastCPUNanos = metric.GetCpuTimeNanos()
	vmUsage.LastMemoryBytes = metric.GetMemoryUsageBytes()
	vmUsage.LastDiskReadBytes = metric.GetDiskReadBytes()
	vmUsage.LastDiskWriteBytes = metric.GetDiskWriteBytes()
	vmUsage.LastNetworkRxBytes = metric.GetNetworkRxBytes()
	vmUsage.LastNetworkTxBytes = metric.GetNetworkTxBytes()
}

// NotifyVMStarted handles VM start notifications
func (a *Aggregator) NotifyVMStarted(vmID, customerID string, startTime int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Initialize or reset VM usage data
	vmUsage := &VMUsageData{ //nolint:exhaustruct // Usage tracking fields are initialized as metrics are received
		VMID:       vmID,
		CustomerID: customerID,
		StartTime:  time.Unix(0, startTime),
	}
	a.vmData[vmID] = vmUsage

	// Track customer mapping
	vmIDs := a.customers[customerID]
	found := false
	for _, existingVMID := range vmIDs {
		if existingVMID == vmID {
			found = true
			break
		}
	}
	if !found {
		a.customers[customerID] = append(vmIDs, vmID)
	}

	a.logger.Info("VM started tracking",
		"vm_id", vmID,
		"customer_id", customerID,
		"start_time", time.Unix(0, startTime).Format(time.RFC3339),
	)
}

// NotifyVMStopped handles VM stop notifications and generates final usage summary
func (a *Aggregator) NotifyVMStopped(vmID string, stopTime int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	vmUsage, exists := a.vmData[vmID]
	if !exists {
		a.logger.Warn("received stop notification for unknown VM", "vm_id", vmID)
		return
	}

	// Generate final usage summary
	endTime := time.Unix(0, stopTime)
	summary := a.generateUsageSummary(vmUsage, endTime)

	a.logger.Info("VM stopped, generating final usage summary",
		"vm_id", vmID,
		"customer_id", vmUsage.CustomerID,
		"stop_time", endTime.Format(time.RFC3339),
		"total_runtime", endTime.Sub(vmUsage.StartTime).String(),
	)

	// Send summary if callback is set
	if a.onUsageSummary != nil {
		a.onUsageSummary(summary)
	}

	// Clean up VM data
	delete(a.vmData, vmID)

	// Remove from customer mapping
	if vmIDs, exists := a.customers[vmUsage.CustomerID]; exists {
		for i, existingVMID := range vmIDs {
			if existingVMID == vmID {
				a.customers[vmUsage.CustomerID] = append(vmIDs[:i], vmIDs[i+1:]...)
				break
			}
		}
	}
}

// GeneratePeriodicSummaries generates usage summaries for all active VMs
func (a *Aggregator) GeneratePeriodicSummaries() {
	a.mu.RLock()
	defer a.mu.RUnlock()

	now := time.Now()

	for vmID, vmUsage := range a.vmData {
		// Skip VMs with no recent activity
		if now.Sub(vmUsage.LastUpdate) > a.aggregationInterval*2 {
			continue
		}

		summary := a.generateUsageSummary(vmUsage, now)

		a.logger.Debug("generated periodic usage summary",
			"vm_id", vmID,
			"customer_id", vmUsage.CustomerID,
			"cpu_time_ms", summary.CPUTimeUsedMs,
			"avg_memory_mb", summary.AvgMemoryUsageBytes/(1024*1024),
			"resource_score", summary.ResourceScore,
		)

		if a.onUsageSummary != nil {
			a.onUsageSummary(summary)
		}
	}
}

// generateUsageSummary creates a usage summary for a VM
func (a *Aggregator) generateUsageSummary(vmUsage *VMUsageData, endTime time.Time) *UsageSummary {
	period := endTime.Sub(vmUsage.StartTime)

	// AIDEV-BUSINESS_RULE: Resource Score Calculation for VM Billing
	// The resource score is a composite metric that combines CPU, memory, and I/O usage
	// into a single billing unit. This weighted formula reflects the relative cost impact
	// of each resource type on infrastructure expenses:
	//
	// 1. CPU Weight (1.0): Highest weight as CPU time directly correlates with compute costs
	//    and represents actual work performed vs. allocated but unused resources
	// 2. Memory Weight (0.5): Medium weight as memory allocation has moderate cost impact
	//    but is often over-provisioned relative to actual usage
	// 3. I/O Weight (0.3): Lower weight as disk I/O has less direct cost impact than CPU/memory
	//    but still represents meaningful resource consumption
	//
	// Formula: resourceScore = (cpuSeconds * 1.0) + (memoryGB * 0.5) + (diskMB * 0.3)
	// These weights should be periodically reviewed against actual infrastructure costs
	// and may need adjustment based on provider pricing changes or workload patterns
	cpuWeight := 1.0
	memoryWeight := 0.5
	ioWeight := 0.3

	cpuScore := float64(vmUsage.TotalCPUNanos) / float64(time.Second) * cpuWeight
	memoryScore := float64(vmUsage.TotalMemoryBytes) / (1024 * 1024 * 1024) * memoryWeight                // GB
	ioScore := float64(vmUsage.TotalDiskReadBytes+vmUsage.TotalDiskWriteBytes) / (1024 * 1024) * ioWeight // MB

	resourceScore := cpuScore + memoryScore + ioScore

	return &UsageSummary{
		VMID:                vmUsage.VMID,
		CustomerID:          vmUsage.CustomerID,
		Period:              period,
		StartTime:           vmUsage.StartTime,
		EndTime:             endTime,
		CPUTimeUsedNanos:    vmUsage.TotalCPUNanos,
		CPUTimeUsedMs:       vmUsage.TotalCPUNanos / 1_000_000,
		AvgMemoryUsageBytes: vmUsage.TotalMemoryBytes,
		MaxMemoryUsageBytes: vmUsage.TotalMemoryBytes,
		DiskReadBytes:       vmUsage.TotalDiskReadBytes,
		DiskWriteBytes:      vmUsage.TotalDiskWriteBytes,
		TotalDiskIO:         vmUsage.TotalDiskReadBytes + vmUsage.TotalDiskWriteBytes,
		NetworkRxBytes:      vmUsage.TotalNetworkRxBytes,
		NetworkTxBytes:      vmUsage.TotalNetworkTxBytes,
		TotalNetworkIO:      vmUsage.TotalNetworkRxBytes + vmUsage.TotalNetworkTxBytes,
		ResourceScore:       resourceScore,
		SampleCount:         vmUsage.SampleCount,
	}
}

// GetCustomerStats returns usage statistics by customer
func (a *Aggregator) GetCustomerStats() map[string]int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := make(map[string]int)
	for customerID, vmIDs := range a.customers {
		stats[customerID] = len(vmIDs)
	}
	return stats
}

// GetActiveVMCount returns the number of currently tracked VMs
func (a *Aggregator) GetActiveVMCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return len(a.vmData)
}

// StartPeriodicAggregation starts the periodic aggregation goroutine
func (a *Aggregator) StartPeriodicAggregation(ctx context.Context) {
	ticker := time.NewTicker(a.aggregationInterval)
	defer ticker.Stop()

	a.logger.InfoContext(ctx, "started periodic aggregation",
		"interval", a.aggregationInterval.String(),
	)

	for {
		select {
		case <-ticker.C:
			a.GeneratePeriodicSummaries()
		case <-ctx.Done():
			a.logger.InfoContext(ctx, "stopping periodic aggregation")
			return
		}
	}
}
