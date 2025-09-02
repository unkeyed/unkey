package docker

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log/slog"
// 	"time"

// 	"github.com/docker/docker/api/types/container"
// 	"github.com/docker/docker/client"
// 	backendtypes "github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
// 	"go.opentelemetry.io/otel/attribute"
// 	"go.opentelemetry.io/otel/trace"
// )

// // MetricsCollector collects metrics from Docker containers
// type MetricsCollector struct {
// 	logger       *slog.Logger
// 	dockerClient *client.Client
// 	tracer       trace.Tracer
// }

// // NewMetricsCollector creates a new metrics collector
// func NewMetricsCollector(logger *slog.Logger, dockerClient *client.Client, tracer trace.Tracer) *MetricsCollector {
// 	return &MetricsCollector{
// 		logger:       logger.With("component", "docker-metrics"),
// 		dockerClient: dockerClient,
// 		tracer:       tracer,
// 	}
// }

// // CollectMetrics collects metrics for a specific container
// func (mc *MetricsCollector) CollectMetrics(ctx context.Context, containerID string) (*backendtypes.VMMetrics, error) {
// 	ctx, span := mc.tracer.Start(ctx, "metald.docker.collect_metrics",
// 		trace.WithAttributes(attribute.String("container_id", containerID)),
// 	)
// 	defer span.End()

// 	// Get container stats (single read, not streaming)
// 	stats, err := mc.dockerClient.ContainerStats(ctx, containerID, false)
// 	if err != nil {
// 		span.RecordError(err)
// 		return nil, fmt.Errorf("failed to get container stats: %w", err)
// 	}
// 	defer stats.Body.Close()

// 	// Parse stats JSON
// 	var dockerStats container.StatsResponse
// 	if err := json.NewDecoder(stats.Body).Decode(&dockerStats); err != nil {
// 		span.RecordError(err)
// 		return nil, fmt.Errorf("failed to decode container stats: %w", err)
// 	}

// 	// Convert to VM metrics
// 	metrics := mc.convertDockerStatsToVMMetrics(&dockerStats)

// 	mc.logger.DebugContext(ctx, "collected container metrics",
// 		slog.String("container_id", containerID),
// 		slog.Int64("cpu_time_nanos", metrics.CpuTimeNanos),
// 		slog.Int64("memory_usage_bytes", metrics.MemoryUsageBytes),
// 		slog.Int64("disk_read_bytes", metrics.DiskReadBytes),
// 		slog.Int64("disk_write_bytes", metrics.DiskWriteBytes),
// 		slog.Int64("network_rx_bytes", metrics.NetworkRxBytes),
// 		slog.Int64("network_tx_bytes", metrics.NetworkTxBytes),
// 	)

// 	return metrics, nil
// }

// // convertDockerStatsToVMMetrics converts Docker stats to VM metrics format
// func (mc *MetricsCollector) convertDockerStatsToVMMetrics(stats *container.StatsResponse) *backendtypes.VMMetrics {
// 	metrics := &backendtypes.VMMetrics{
// 		Timestamp:        time.Now(),
// 		CpuTimeNanos:     0,
// 		MemoryUsageBytes: 0,
// 		DiskReadBytes:    0,
// 		DiskWriteBytes:   0,
// 		NetworkRxBytes:   0,
// 		NetworkTxBytes:   0,
// 	}

// 	// CPU metrics
// 	if stats.CPUStats.CPUUsage.TotalUsage > 0 {
// 		metrics.CpuTimeNanos = safeUint64ToInt64(stats.CPUStats.CPUUsage.TotalUsage)
// 	}

// 	// Memory metrics
// 	if stats.MemoryStats.Usage > 0 {
// 		metrics.MemoryUsageBytes = safeUint64ToInt64(stats.MemoryStats.Usage)
// 	}

// 	// Disk I/O metrics
// 	if stats.BlkioStats.IoServiceBytesRecursive != nil {
// 		for _, blkio := range stats.BlkioStats.IoServiceBytesRecursive {
// 			switch blkio.Op {
// 			case "Read":
// 				metrics.DiskReadBytes += safeUint64ToInt64(blkio.Value)
// 			case "Write":
// 				metrics.DiskWriteBytes += safeUint64ToInt64(blkio.Value)
// 			}
// 		}
// 	}

// 	// Network I/O metrics
// 	if stats.Networks != nil {
// 		for _, netStats := range stats.Networks {
// 			metrics.NetworkRxBytes += safeUint64ToInt64(netStats.RxBytes)
// 			metrics.NetworkTxBytes += safeUint64ToInt64(netStats.TxBytes)
// 		}
// 	}

// 	return metrics
// }

// // CollectBulkMetrics collects metrics for multiple containers
// func (mc *MetricsCollector) CollectBulkMetrics(ctx context.Context, containerIDs []string) (map[string]*backendtypes.VMMetrics, error) {
// 	ctx, span := mc.tracer.Start(ctx, "metald.docker.collect_bulk_metrics",
// 		trace.WithAttributes(attribute.Int("container_count", len(containerIDs))),
// 	)
// 	defer span.End()

// 	results := make(map[string]*backendtypes.VMMetrics)

// 	for _, containerID := range containerIDs {
// 		metrics, err := mc.CollectMetrics(ctx, containerID)
// 		if err != nil {
// 			mc.logger.WarnContext(ctx, "failed to collect metrics for container",
// 				slog.String("container_id", containerID),
// 				slog.String("error", err.Error()),
// 			)
// 			continue
// 		}
// 		results[containerID] = metrics
// 	}

// 	span.SetAttributes(attribute.Int("successful_collections", len(results)))
// 	return results, nil
// }

// // StreamMetrics streams metrics for a container (for real-time monitoring)
// func (mc *MetricsCollector) StreamMetrics(ctx context.Context, containerID string, interval time.Duration) (<-chan *backendtypes.VMMetrics, <-chan error) {
// 	metricsChan := make(chan *backendtypes.VMMetrics, 1)
// 	errorChan := make(chan error, 1)

// 	go func() {
// 		defer close(metricsChan)
// 		defer close(errorChan)

// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-ctx.Done():
// 				return
// 			case <-ticker.C:
// 				metrics, err := mc.CollectMetrics(ctx, containerID)
// 				if err != nil {
// 					select {
// 					case errorChan <- err:
// 					case <-ctx.Done():
// 						return
// 					}
// 					continue
// 				}

// 				select {
// 				case metricsChan <- metrics:
// 				case <-ctx.Done():
// 					return
// 				}
// 			}
// 		}
// 	}()

// 	return metricsChan, errorChan
// }

// // CalculateCPUPercent calculates CPU percentage from Docker stats
// func (mc *MetricsCollector) CalculateCPUPercent(current, previous *container.StatsResponse) float64 {
// 	if current == nil || previous == nil {
// 		return 0.0
// 	}

// 	// Calculate CPU delta
// 	cpuDelta := float64(current.CPUStats.CPUUsage.TotalUsage - previous.CPUStats.CPUUsage.TotalUsage)

// 	// Calculate system CPU delta
// 	systemDelta := float64(current.CPUStats.SystemUsage - previous.CPUStats.SystemUsage)

// 	// Calculate number of CPUs
// 	onlineCPUs := float64(current.CPUStats.OnlineCPUs)
// 	if onlineCPUs == 0 {
// 		onlineCPUs = float64(len(current.CPUStats.CPUUsage.PercpuUsage))
// 	}

// 	if systemDelta > 0 && cpuDelta > 0 {
// 		return (cpuDelta / systemDelta) * onlineCPUs * 100.0
// 	}

// 	return 0.0
// }

// // CalculateMemoryPercent calculates memory percentage from Docker stats
// func (mc *MetricsCollector) CalculateMemoryPercent(stats *container.StatsResponse) float64 {
// 	if stats == nil || stats.MemoryStats.Limit == 0 {
// 		return 0.0
// 	}

// 	// Calculate memory usage percentage
// 	usage := float64(stats.MemoryStats.Usage)
// 	limit := float64(stats.MemoryStats.Limit)

// 	return (usage / limit) * 100.0
// }

// // CalculateNetworkIORate calculates network I/O rate from Docker stats
// func (mc *MetricsCollector) CalculateNetworkIORate(current, previous *container.StatsResponse, timeDelta time.Duration) (rxRate, txRate float64) {
// 	if current == nil || previous == nil || timeDelta == 0 {
// 		return 0.0, 0.0
// 	}

// 	var currentRx, currentTx, previousRx, previousTx uint64

// 	// Sum network stats from all interfaces
// 	for _, netStats := range current.Networks {
// 		currentRx += netStats.RxBytes
// 		currentTx += netStats.TxBytes
// 	}

// 	for _, netStats := range previous.Networks {
// 		previousRx += netStats.RxBytes
// 		previousTx += netStats.TxBytes
// 	}

// 	// Calculate rates (bytes per second)
// 	seconds := timeDelta.Seconds()
// 	rxRate = float64(currentRx-previousRx) / seconds
// 	txRate = float64(currentTx-previousTx) / seconds

// 	return rxRate, txRate
// }

// // CalculateBlockIORate calculates block I/O rate from Docker stats
// func (mc *MetricsCollector) CalculateBlockIORate(current, previous *container.StatsResponse, timeDelta time.Duration) (readRate, writeRate float64) {
// 	if current == nil || previous == nil || timeDelta == 0 {
// 		return 0.0, 0.0
// 	}

// 	var currentRead, currentWrite, previousRead, previousWrite uint64

// 	// Sum block I/O stats
// 	for _, blkio := range current.BlkioStats.IoServiceBytesRecursive {
// 		switch blkio.Op {
// 		case "Read":
// 			currentRead += blkio.Value
// 		case "Write":
// 			currentWrite += blkio.Value
// 		}
// 	}

// 	for _, blkio := range previous.BlkioStats.IoServiceBytesRecursive {
// 		switch blkio.Op {
// 		case "Read":
// 			previousRead += blkio.Value
// 		case "Write":
// 			previousWrite += blkio.Value
// 		}
// 	}

// 	// Calculate rates (bytes per second)
// 	seconds := timeDelta.Seconds()
// 	readRate = float64(currentRead-previousRead) / seconds
// 	writeRate = float64(currentWrite-previousWrite) / seconds

// 	return readRate, writeRate
// }

// // GetContainerResourceLimits gets resource limits for a container
// func (mc *MetricsCollector) GetContainerResourceLimits(ctx context.Context, containerID string) (*ResourceLimits, error) {
// 	ctx, span := mc.tracer.Start(ctx, "metald.docker.get_resource_limits",
// 		trace.WithAttributes(attribute.String("container_id", containerID)),
// 	)
// 	defer span.End()

// 	// Inspect container to get resource limits
// 	inspect, err := mc.dockerClient.ContainerInspect(ctx, containerID)
// 	if err != nil {
// 		span.RecordError(err)
// 		return nil, fmt.Errorf("failed to inspect container: %w", err)
// 	}

// 	limits := &ResourceLimits{
// 		Memory:   inspect.HostConfig.Memory,
// 		NanoCPUs: inspect.HostConfig.NanoCPUs,
// 	}

// 	return limits, nil
// }

// // ResourceLimits represents resource limits for a container
// type ResourceLimits struct {
// 	Memory   int64 // Memory limit in bytes
// 	NanoCPUs int64 // CPU limit in nano CPUs
// }

// // GetCPULimit returns CPU limit as number of CPUs
// func (rl *ResourceLimits) GetCPULimit() float64 {
// 	return float64(rl.NanoCPUs) / 1e9
// }

// // GetMemoryLimit returns memory limit in bytes
// func (rl *ResourceLimits) GetMemoryLimit() int64 {
// 	return rl.Memory
// }

// // AIDEV-NOTE: Docker metrics collection provides comprehensive monitoring
// // capabilities for containers treated as VMs. Key features:
// // 1. Real-time metrics collection via Docker stats API
// // 2. CPU, memory, disk I/O, and network I/O monitoring
// // 3. Streaming metrics for continuous monitoring
// // 4. Resource limit awareness for accurate percentage calculations
// // 5. Bulk metrics collection for efficient monitoring of multiple containers
