package billing

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/observability"
)

// MetricsCollector manages high-frequency metrics collection for billing
type MetricsCollector struct {
	backend        types.Backend
	billingClient  BillingClient
	logger         *slog.Logger
	billingMetrics *observability.BillingMetrics

	// State management
	mu        sync.RWMutex
	activeVMs map[string]*VMMetricsTracker

	// Configuration
	collectionInterval time.Duration
	batchSize          int
	instanceID         string
}

// VMMetricsTracker tracks metrics collection for a single VM
type VMMetricsTracker struct {
	vmID       string
	customerID string
	startTime  time.Time
	lastSent   time.Time
	buffer     []*types.VMMetrics
	ticker     *time.Ticker
	stopCh     chan struct{}
	doneCh     chan struct{} // Signals when goroutine has completely stopped
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex

	// Error tracking
	consecutiveErrors int
	lastError         time.Time
}

// NewMetricsCollector creates a new metrics collector instance
func NewMetricsCollector(backend types.Backend, billingClient BillingClient, logger *slog.Logger, instanceID string, billingMetrics *observability.BillingMetrics) *MetricsCollector {
	//exhaustruct:ignore
	return &MetricsCollector{
		backend:            backend,
		billingClient:      billingClient,
		logger:             logger.With("component", "metrics_collector"),
		billingMetrics:     billingMetrics,
		activeVMs:          make(map[string]*VMMetricsTracker),
		collectionInterval: 5 * time.Minute,
		batchSize:          1, // Very small batch size for 5min intervals
		instanceID:         instanceID,
	}
}

// StartCollection begins metrics collection for a VM
func (mc *MetricsCollector) StartCollection(vmID, customerID string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.activeVMs[vmID]; exists {
		return fmt.Errorf("metrics collection already active for vm %s", vmID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	//exhaustruct:ignore
	tracker := &VMMetricsTracker{
		vmID:       vmID,
		customerID: customerID,
		startTime:  time.Now(),
		lastSent:   time.Now(),
		buffer:     make([]*types.VMMetrics, 0, mc.batchSize),
		ticker:     time.NewTicker(mc.collectionInterval),
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
		ctx:        ctx,
		cancel:     cancel,
	}

	mc.activeVMs[vmID] = tracker

	// Notify billaged that VM started
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := mc.billingClient.NotifyVmStarted(ctx, vmID, customerID, tracker.startTime.UnixNano()); err != nil {
			mc.logger.Error("failed to notify VM started",
				"vm_id", vmID,
				"error", err,
			)
		}
	}()

	// Start collection goroutine
	go mc.runCollection(tracker)

	mc.logger.Info("started metrics collection",
		"vm_id", vmID,
		"customer_id", customerID,
		"interval", mc.collectionInterval,
	)

	return nil
}

// StopCollection stops metrics collection for a VM with proper timeout and cleanup
func (mc *MetricsCollector) StopCollection(vmID string) {
	mc.mu.Lock()
	tracker, exists := mc.activeVMs[vmID]
	if !exists {
		mc.mu.Unlock()
		mc.logger.Debug("metrics collection not active for vm", "vm_id", vmID)
		return
	}
	delete(mc.activeVMs, vmID)
	mc.mu.Unlock()

	mc.logger.Info("stopping metrics collection", "vm_id", vmID)

	// Cancel the context to interrupt any blocking operations
	tracker.cancel()

	// Signal stop to the collection goroutine
	close(tracker.stopCh)

	// Wait for the goroutine to finish with a timeout
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	select {
	case <-tracker.doneCh:
		mc.logger.Debug("metrics collection goroutine stopped gracefully", "vm_id", vmID)
	case <-timeout.C:
		mc.logger.Warn("metrics collection goroutine did not stop within timeout",
			"vm_id", vmID,
			"timeout", "5s")
	}

	// Notify billaged that VM stopped
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := mc.billingClient.NotifyVmStopped(ctx, vmID, time.Now().UnixNano()); err != nil {
			mc.logger.Error("failed to notify VM stopped",
				"vm_id", vmID,
				"error", err,
			)
		}
	}()

	mc.logger.Info("stopped metrics collection",
		"vm_id", vmID,
		"duration", time.Since(tracker.startTime),
	)
}

// GetActiveVMs returns a list of VMs currently being tracked
func (mc *MetricsCollector) GetActiveVMs() []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	vms := make([]string, 0, len(mc.activeVMs))
	for vmID := range mc.activeVMs {
		vms = append(vms, vmID)
	}
	return vms
}

// StartHeartbeat begins sending periodic heartbeats to billaged
func (mc *MetricsCollector) StartHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			activeVMs := mc.GetActiveVMs()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := mc.billingClient.SendHeartbeat(ctx, mc.instanceID, activeVMs)
			cancel()

			if err != nil {
				mc.logger.Error("heartbeat failed",
					"instance_id", mc.instanceID,
					"active_vms_count", len(activeVMs),
					"error", err,
				)
			} else {
				// Record successful heartbeat
				if mc.billingMetrics != nil {
					mc.billingMetrics.RecordHeartbeatSent(ctx, mc.instanceID)
				}
				mc.logger.Debug("heartbeat sent successfully",
					"instance_id", mc.instanceID,
					"active_vms_count", len(activeVMs),
				)
			}
		}
	}()

	mc.logger.Info("started heartbeat service",
		"instance_id", mc.instanceID,
		"interval", "30s",
	)
}

// runCollection performs the metrics collection loop for a single VM
func (mc *MetricsCollector) runCollection(tracker *VMMetricsTracker) {
	defer func() {
		tracker.ticker.Stop()
		close(tracker.doneCh) // Signal that goroutine has completed
	}()

	for {
		select {
		case <-tracker.ctx.Done():
			// Context cancelled - stop immediately
			mc.logger.Debug("metrics collection context cancelled", "vm_id", tracker.vmID)
			return
		case <-tracker.ticker.C:
			// Collect metrics with cancellable context and timeout
			start := time.Now()
			ctx, cancel := context.WithTimeout(tracker.ctx, 2*time.Second)
			metrics, err := mc.backend.GetVMMetrics(ctx, tracker.vmID)
			cancel()
			collectDuration := time.Since(start)

			// Record VM metrics request
			if mc.billingMetrics != nil {
				mc.billingMetrics.RecordVMMetricsRequest(ctx, tracker.vmID)
			}

			if err != nil {
				tracker.consecutiveErrors++
				tracker.lastError = time.Now()

				mc.logger.Error("failed to collect metrics",
					"vm_id", tracker.vmID,
					"consecutive_errors", tracker.consecutiveErrors,
					"error", err,
				)

				// Skip this collection cycle but continue
				continue
			}

			// Reset error tracking on success
			if tracker.consecutiveErrors > 0 {
				mc.logger.Info("metrics collection recovered",
					"vm_id", tracker.vmID,
					"previous_errors", tracker.consecutiveErrors,
				)
				tracker.consecutiveErrors = 0
			}

			tracker.mu.Lock()
			tracker.buffer = append(tracker.buffer, metrics)

			// Record metrics collected
			if mc.billingMetrics != nil {
				mc.billingMetrics.RecordMetricsCollected(ctx, tracker.vmID, 1, collectDuration)
			}

			mc.logger.Debug("collected metrics",
				"vm_id", tracker.vmID,
				"collect_duration_ms", collectDuration.Milliseconds(),
				"buffer_size", len(tracker.buffer),
				"cpu_time_nanos", metrics.CpuTimeNanos,
				"memory_bytes", metrics.MemoryUsageBytes,
			)

			// Send batch when full
			if len(tracker.buffer) >= mc.batchSize {
				mc.sendBatch(tracker)
				tracker.buffer = tracker.buffer[:0] // Reset buffer
				tracker.lastSent = time.Now()
			}
			tracker.mu.Unlock()

		case <-tracker.stopCh:
			// Send final batch
			tracker.mu.Lock()
			if len(tracker.buffer) > 0 {
				mc.logger.Info("sending final metrics batch",
					"vm_id", tracker.vmID,
					"final_batch_size", len(tracker.buffer),
				)
				mc.sendBatch(tracker)
			}
			tracker.mu.Unlock()
			return
		}
	}
}

// sendBatch sends a batch of metrics to billaged
func (mc *MetricsCollector) sendBatch(tracker *VMMetricsTracker) {
	if len(tracker.buffer) == 0 {
		return
	}

	batchCopy := make([]*types.VMMetrics, len(tracker.buffer))
	copy(batchCopy, tracker.buffer)

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := mc.billingClient.SendMetricsBatch(ctx, tracker.vmID, tracker.customerID, batchCopy)
	sendDuration := time.Since(start)

	if err != nil {
		mc.logger.Error("failed to send metrics batch",
			"vm_id", tracker.vmID,
			"customer_id", tracker.customerID,
			"batch_size", len(batchCopy),
			"send_duration_ms", sendDuration.Milliseconds(),
			"error", err,
		)
		// TODO: Implement retry logic with local queuing
		return
	}

	// Record successful batch send
	if mc.billingMetrics != nil {
		mc.billingMetrics.RecordBillingBatchSent(ctx, tracker.vmID, tracker.customerID, len(batchCopy), sendDuration)
	}

	mc.logger.Debug("sent metrics batch successfully",
		"vm_id", tracker.vmID,
		"customer_id", tracker.customerID,
		"batch_size", len(batchCopy),
		"send_duration_ms", sendDuration.Milliseconds(),
	)
}
