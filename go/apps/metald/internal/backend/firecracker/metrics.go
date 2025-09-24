//go:build linux
// +build linux

package firecracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GetVMMetrics retrieves metrics for a specific VM
func (c *Client) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.get_vm_metrics",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	c.logger.LogAttrs(ctx, slog.LevelDebug, "retrieving VM metrics",
		slog.String("vm_id", vmID),
	)

	// Get the VM from registry
	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return nil, err
	}

	// Check if VM has a machine instance
	if vm.Machine == nil {
		err := fmt.Errorf("vm %s has no firecracker process", vmID)
		span.RecordError(err)
		return nil, err
	}

	// Calculate the jailer root path
	jailerRoot := filepath.Join(
		c.jailerConfig.ChrootBaseDir,
		"firecracker",
		vmID,
		"root",
	)

	// Read metrics from the FIFO
	metricsPath := filepath.Join(jailerRoot, "metrics.fifo")
	metrics, err := c.readFirecrackerMetrics(ctx, metricsPath)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	return metrics, nil
}

// readFirecrackerMetrics reads metrics from Firecracker's metrics FIFO
func (c *Client) readFirecrackerMetrics(ctx context.Context, metricsPath string) (*types.VMMetrics, error) {
	c.logger.LogAttrs(ctx, slog.LevelDebug, "reading firecracker metrics",
		slog.String("metrics_path", metricsPath),
	)

	// Open the metrics FIFO (non-blocking read)
	file, err := os.OpenFile(metricsPath, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open metrics FIFO: %w", err)
	}
	defer file.Close()

	// Read all available data from the FIFO
	data, err := io.ReadAll(file)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read metrics data: %w", err)
	}

	// If no data available, return empty metrics
	if len(data) == 0 {
		c.logger.LogAttrs(ctx, slog.LevelDebug, "no metrics data available",
			slog.String("metrics_path", metricsPath),
		)
		return &types.VMMetrics{}, nil
	}

	// Parse the JSON metrics
	var rawMetrics map[string]interface{}
	if err := json.Unmarshal(data, &rawMetrics); err != nil {
		return nil, fmt.Errorf("failed to parse metrics JSON: %w", err)
	}

	// Convert raw metrics to structured format
	metrics := c.parseRawMetrics(rawMetrics)

	c.logger.LogAttrs(ctx, slog.LevelDebug, "successfully read VM metrics",
		slog.Int64("cpu_time_ns", metrics.CpuTimeNanos),
		slog.Int64("memory_usage_bytes", metrics.MemoryUsageBytes),
		slog.Int64("disk_read_bytes", metrics.DiskReadBytes),
		slog.Int64("disk_write_bytes", metrics.DiskWriteBytes),
		slog.Int64("network_rx_bytes", metrics.NetworkRxBytes),
		slog.Int64("network_tx_bytes", metrics.NetworkTxBytes),
	)

	return metrics, nil
}

// parseRawMetrics converts raw Firecracker metrics to structured format
func (c *Client) parseRawMetrics(raw map[string]interface{}) *types.VMMetrics {
	metrics := &types.VMMetrics{}

	// Extract CPU metrics
	if cpu, ok := raw["cpu"].(map[string]interface{}); ok {
		if cpuTimeNs, ok := cpu["cpu_time_ms"].(float64); ok {
			metrics.CpuTimeNanos = int64(cpuTimeNs)
		}
	}

	// Extract memory metrics
	if memory, ok := raw["memory"].(map[string]interface{}); ok {
		if memUsageBytes, ok := memory["memory_usage_bytes"].(float64); ok {
			metrics.MemoryUsageBytes = int64(memUsageBytes)
		}
	}

	// Extract disk metrics
	if disk, ok := raw["disk"].(map[string]interface{}); ok {
		if readBytes, ok := disk["read_bytes"].(float64); ok {
			metrics.DiskReadBytes = int64(readBytes)
		}
		if writeBytes, ok := disk["write_bytes"].(float64); ok {
			metrics.DiskWriteBytes = int64(writeBytes)
		}
	}

	// Extract network metrics
	if network, ok := raw["network"].(map[string]interface{}); ok {
		if rxBytes, ok := network["rx_bytes"].(float64); ok {
			metrics.NetworkRxBytes = int64(rxBytes)
		}
		if txBytes, ok := network["tx_bytes"].(float64); ok {
			metrics.NetworkTxBytes = int64(txBytes)
		}
	}

	return metrics
}
