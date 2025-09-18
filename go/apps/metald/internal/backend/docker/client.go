package docker

// import (
// 	"context"
// 	"crypto/rand"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"math/big"
// 	"strconv"
// 	"strings"
// 	"sync"
// 	"time"

// 	"github.com/docker/docker/api/types/container"
// 	"github.com/docker/docker/api/types/image"
// 	"github.com/docker/docker/client"
// 	"github.com/docker/go-connections/nat"
// 	backendtypes "github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
// 	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
// 	"go.opentelemetry.io/otel"
// 	"go.opentelemetry.io/otel/attribute"
// 	"go.opentelemetry.io/otel/metric"
// 	"go.opentelemetry.io/otel/trace"
// )

// // safeUint64ToInt64 safely converts uint64 to int64, clamping to max int64 on overflow
// func safeUint64ToInt64(val uint64) int64 {
// 	const maxInt64 = int64(^uint64(0) >> 1) // 2^63 - 1
// 	if val > uint64(maxInt64) {
// 		return maxInt64 // Clamp to max int64 instead of overflowing
// 	}
// 	return int64(val)
// }

// // safeIntToInt32 safely converts int to int32, clamping to int32 bounds on overflow
// func safeIntToInt32(val int) int32 {
// 	const maxInt32 = 2147483647  // 2^31 - 1
// 	const minInt32 = -2147483648 // -2^31
// 	if val > maxInt32 {
// 		return maxInt32
// 	}
// 	if val < minInt32 {
// 		return minInt32
// 	}
// 	return int32(val)
// }

// // DockerBackend implements the Backend interface using Docker containers
// type DockerBackend struct {
// 	logger          *slog.Logger
// 	dockerClient    *client.Client
// 	config          *DockerBackendConfig
// 	vmRegistry      map[string]*dockerVM
// 	portAllocator   *portAllocator
// 	mutex           sync.RWMutex
// 	tracer          trace.Tracer
// 	meter           metric.Meter
// 	vmCreateCounter metric.Int64Counter
// 	vmDeleteCounter metric.Int64Counter
// 	vmBootCounter   metric.Int64Counter
// 	vmErrorCounter  metric.Int64Counter
// }

// // portAllocator manages port allocation for containers
// type portAllocator struct {
// 	mutex     sync.Mutex
// 	allocated map[int]string // port -> vmID
// 	minPort   int
// 	maxPort   int
// }

// // NewDockerBackend creates a new Docker backend
// func NewDockerBackend(logger *slog.Logger, config *DockerBackendConfig) (*DockerBackend, error) {
// 	if config == nil {
// 		config = DefaultDockerBackendConfig()
// 	}

// 	// Create Docker client
// 	dockerClient, err := client.NewClientWithOpts(
// 		client.FromEnv,
// 		client.WithAPIVersionNegotiation(),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create Docker client: %w", err)
// 	}

// 	// Verify Docker connection
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	if _, pingErr := dockerClient.Ping(ctx); pingErr != nil {
// 		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", pingErr)
// 	}

// 	// Set up telemetry
// 	tracer := otel.Tracer("metald.docker.backend")
// 	meter := otel.Meter("metald.docker.backend")

// 	vmCreateCounter, err := meter.Int64Counter("vm_create_total",
// 		metric.WithDescription("Total number of VM create operations"),
// 		metric.WithUnit("1"),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create vm_create counter: %w", err)
// 	}

// 	vmDeleteCounter, err := meter.Int64Counter("vm_delete_total",
// 		metric.WithDescription("Total number of VM delete operations"),
// 		metric.WithUnit("1"),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create vm_delete counter: %w", err)
// 	}

// 	vmBootCounter, err := meter.Int64Counter("vm_boot_total",
// 		metric.WithDescription("Total number of VM boot operations"),
// 		metric.WithUnit("1"),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create vm_boot counter: %w", err)
// 	}

// 	vmErrorCounter, err := meter.Int64Counter("vm_error_total",
// 		metric.WithDescription("Total number of VM operation errors"),
// 		metric.WithUnit("1"),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create vm_error counter: %w", err)
// 	}

// 	// Create port allocator
// 	portAllocator := &portAllocator{
// 		allocated: make(map[int]string),
// 		minPort:   config.PortRange.Min,
// 		maxPort:   config.PortRange.Max,
// 	}

// 	backend := &DockerBackend{
// 		logger:          logger.With("backend", "docker"),
// 		dockerClient:    dockerClient,
// 		config:          config,
// 		vmRegistry:      make(map[string]*dockerVM),
// 		portAllocator:   portAllocator,
// 		tracer:          tracer,
// 		meter:           meter,
// 		vmCreateCounter: vmCreateCounter,
// 		vmDeleteCounter: vmDeleteCounter,
// 		vmBootCounter:   vmBootCounter,
// 		vmErrorCounter:  vmErrorCounter,
// 	}

// 	return backend, nil
// }

// // CreateVM creates a new Docker container representing a VM
// func (d *DockerBackend) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.create_vm",
// 		trace.WithAttributes(
// 			attribute.Int("vcpus", int(config.GetVcpuCount())),
// 			attribute.Int64("memory_bytes", int64(config.GetMemorySizeMib())),
// 		),
// 	)
// 	defer span.End()

// 	// Generate VM ID
// 	vmID := d.generateVMID()
// 	span.SetAttributes(attribute.String("vm_id", vmID))

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "creating VM with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.Int("vcpus", int(config.GetVcpuCount())),
// 		slog.Int64("memory_bytes", int64(config.GetMemorySizeMib())),
// 	)

// 	// Convert VM config to container spec
// 	containerSpec, err := d.vmConfigToContainerSpec(ctx, vmID, config)
// 	if err != nil {
// 		span.RecordError(err)
// 		d.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
// 			attribute.String("operation", "create"),
// 			attribute.String("error", "config_conversion"),
// 		))
// 		return "", fmt.Errorf("failed to convert VM config to container spec: %w", err)
// 	}

// 	// Create container
// 	containerID, err := d.createContainer(ctx, containerSpec)
// 	if err != nil {
// 		span.RecordError(err)
// 		d.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
// 			attribute.String("operation", "create"),
// 			attribute.String("error", "container_creation"),
// 		))
// 		return "", fmt.Errorf("failed to create container: %w", err)
// 	}

// 	// Register VM
// 	d.mutex.Lock()
// 	vm := &dockerVM{
// 		ID:           vmID,
// 		ContainerID:  containerID,
// 		Config:       config,
// 		State:        metaldv1.VmState_VM_STATE_CREATED,
// 		PortMappings: containerSpec.PortMappings,
// 		CreatedAt:    time.Now(),
// 	}
// 	d.vmRegistry[vmID] = vm
// 	d.mutex.Unlock()

// 	d.vmCreateCounter.Add(ctx, 1, metric.WithAttributes(
// 		attribute.String("status", "success"),
// 	))

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "VM created successfully with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", containerID),
// 	)

// 	return vmID, nil
// }

// // BootVM starts a Docker container representing a VM
// func (d *DockerBackend) BootVM(ctx context.Context, vmID string) error {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.boot_vm",
// 		trace.WithAttributes(attribute.String("vm_id", vmID)),
// 	)
// 	defer span.End()

// 	d.mutex.Lock()
// 	vm, exists := d.vmRegistry[vmID]
// 	d.mutex.Unlock()

// 	if !exists {
// 		err := fmt.Errorf("vm %s not found", vmID)
// 		span.RecordError(err)
// 		d.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
// 			attribute.String("operation", "boot"),
// 			attribute.String("error", "vm_not_found"),
// 		))
// 		return err
// 	}

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "booting VM with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	// Check if container still exists before starting
// 	_, err := d.dockerClient.ContainerInspect(ctx, vm.ContainerID)
// 	if err != nil {
// 		span.RecordError(err)
// 		d.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
// 			attribute.String("operation", "boot"),
// 			attribute.String("error", "container_not_found"),
// 		))
// 		return fmt.Errorf("container not found before start: %w", err)
// 	}

// 	// Start container
// 	if startErr := d.dockerClient.ContainerStart(ctx, vm.ContainerID, container.StartOptions{}); startErr != nil {
// 		span.RecordError(startErr)
// 		d.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
// 			attribute.String("operation", "boot"),
// 			attribute.String("error", "container_start"),
// 		))
// 		return fmt.Errorf("failed to start container: %w", startErr)
// 	}

// 	// Update VM state and network info
// 	d.mutex.Lock()
// 	vm.State = metaldv1.VmState_VM_STATE_RUNNING

// 	// Get container network info
// 	// networkInfo, err := d.getContainerNetworkInfo(ctx, vm.ContainerID)
// 	if err != nil {
// 		d.logger.WarnContext(ctx, "failed to get container network info",
// 			"vm_id", vmID,
// 			"container_id", vm.ContainerID,
// 			"error", err,
// 		)
// 	} else {
// 		// vm.NetworkInfo = networkInfo
// 	}
// 	d.mutex.Unlock()

// 	d.vmBootCounter.Add(ctx, 1, metric.WithAttributes(
// 		attribute.String("status", "success"),
// 	))

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "VM booted successfully with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	return nil
// }

// // DeleteVM removes a Docker container representing a VM
// func (d *DockerBackend) DeleteVM(ctx context.Context, vmID string) error {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.delete_vm",
// 		trace.WithAttributes(attribute.String("vm_id", vmID)),
// 	)
// 	defer span.End()

// 	d.mutex.Lock()
// 	vm, exists := d.vmRegistry[vmID]
// 	if !exists {
// 		d.mutex.Unlock()
// 		err := fmt.Errorf("vm %s not found", vmID)
// 		span.RecordError(err)
// 		d.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
// 			attribute.String("operation", "delete"),
// 			attribute.String("error", "vm_not_found"),
// 		))
// 		return err
// 	}

// 	// Release allocated ports
// 	for _, mapping := range vm.PortMappings {
// 		d.portAllocator.releasePort(mapping.HostPort, vmID)
// 	}

// 	delete(d.vmRegistry, vmID)
// 	d.mutex.Unlock()

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "deleting VM with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	// Remove container (force remove)
// 	if err := d.dockerClient.ContainerRemove(ctx, vm.ContainerID, container.RemoveOptions{
// 		Force: true,
// 	}); err != nil {
// 		span.RecordError(err)
// 		d.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
// 			attribute.String("operation", "delete"),
// 			attribute.String("error", "container_removal"),
// 		))
// 		return fmt.Errorf("failed to remove container: %w", err)
// 	}

// 	d.vmDeleteCounter.Add(ctx, 1, metric.WithAttributes(
// 		attribute.String("status", "success"),
// 	))

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "VM deleted successfully with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	return nil
// }

// // ShutdownVM gracefully shuts down a Docker container
// func (d *DockerBackend) ShutdownVM(ctx context.Context, vmID string) error {
// 	return d.ShutdownVMWithOptions(ctx, vmID, false, 30)
// }

// // ShutdownVMWithOptions shuts down a Docker container with options
// func (d *DockerBackend) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.shutdown_vm",
// 		trace.WithAttributes(
// 			attribute.String("vm_id", vmID),
// 			attribute.Bool("force", force),
// 			attribute.Int("timeout_seconds", int(timeoutSeconds)),
// 		),
// 	)
// 	defer span.End()

// 	d.mutex.Lock()
// 	vm, exists := d.vmRegistry[vmID]
// 	d.mutex.Unlock()

// 	if !exists {
// 		err := fmt.Errorf("vm %s not found", vmID)
// 		span.RecordError(err)
// 		return err
// 	}

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down VM with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 		slog.Bool("force", force),
// 		slog.Int("timeout_seconds", int(timeoutSeconds)),
// 	)

// 	// Stop container
// 	timeoutInt := int(timeoutSeconds)
// 	if err := d.dockerClient.ContainerStop(ctx, vm.ContainerID, container.StopOptions{Timeout: &timeoutInt}); err != nil {
// 		span.RecordError(err)
// 		return fmt.Errorf("failed to stop container: %w", err)
// 	}

// 	// Update VM state
// 	d.mutex.Lock()
// 	vm.State = metaldv1.VmState_VM_STATE_SHUTDOWN
// 	d.mutex.Unlock()

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "VM shutdown successfully with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	return nil
// }

// // PauseVM pauses a Docker container
// func (d *DockerBackend) PauseVM(ctx context.Context, vmID string) error {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.pause_vm",
// 		trace.WithAttributes(attribute.String("vm_id", vmID)),
// 	)
// 	defer span.End()

// 	d.mutex.Lock()
// 	vm, exists := d.vmRegistry[vmID]
// 	d.mutex.Unlock()

// 	if !exists {
// 		err := fmt.Errorf("vm %s not found", vmID)
// 		span.RecordError(err)
// 		return err
// 	}

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "pausing VM with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	// Pause container
// 	if err := d.dockerClient.ContainerPause(ctx, vm.ContainerID); err != nil {
// 		span.RecordError(err)
// 		return fmt.Errorf("failed to pause container: %w", err)
// 	}

// 	// Update VM state
// 	d.mutex.Lock()
// 	vm.State = metaldv1.VmState_VM_STATE_PAUSED
// 	d.mutex.Unlock()

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "VM paused successfully with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	return nil
// }

// // ResumeVM resumes a paused Docker container
// func (d *DockerBackend) ResumeVM(ctx context.Context, vmID string) error {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.resume_vm",
// 		trace.WithAttributes(attribute.String("vm_id", vmID)),
// 	)
// 	defer span.End()

// 	d.mutex.Lock()
// 	vm, exists := d.vmRegistry[vmID]
// 	d.mutex.Unlock()

// 	if !exists {
// 		err := fmt.Errorf("vm %s not found", vmID)
// 		span.RecordError(err)
// 		return err
// 	}

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "resuming VM with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	// Unpause container
// 	if err := d.dockerClient.ContainerUnpause(ctx, vm.ContainerID); err != nil {
// 		span.RecordError(err)
// 		return fmt.Errorf("failed to unpause container: %w", err)
// 	}

// 	// Update VM state
// 	d.mutex.Lock()
// 	vm.State = metaldv1.VmState_VM_STATE_RUNNING
// 	d.mutex.Unlock()

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "VM resumed successfully with Docker backend",
// 		slog.String("vm_id", vmID),
// 		slog.String("container_id", vm.ContainerID),
// 	)

// 	return nil
// }

// // RebootVM restarts a Docker container
// func (d *DockerBackend) RebootVM(ctx context.Context, vmID string) error {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.reboot_vm",
// 		trace.WithAttributes(attribute.String("vm_id", vmID)),
// 	)
// 	defer span.End()

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "rebooting VM with Docker backend",
// 		slog.String("vm_id", vmID),
// 	)

// 	// Shutdown the VM
// 	if err := d.ShutdownVMWithOptions(ctx, vmID, false, 30); err != nil {
// 		span.RecordError(err)
// 		return fmt.Errorf("failed to shutdown VM for reboot: %w", err)
// 	}

// 	// Wait a moment
// 	time.Sleep(1 * time.Second)

// 	// Boot the VM again
// 	if err := d.BootVM(ctx, vmID); err != nil {
// 		span.RecordError(err)
// 		return fmt.Errorf("failed to boot VM after shutdown: %w", err)
// 	}

// 	d.logger.LogAttrs(ctx, slog.LevelInfo, "VM rebooted successfully with Docker backend",
// 		slog.String("vm_id", vmID),
// 	)

// 	return nil
// }

// // GetVMInfo returns information about a VM
// func (d *DockerBackend) GetVMInfo(ctx context.Context, vmID string) (*backendtypes.VMInfo, error) {
// 	_, span := d.tracer.Start(ctx, "metald.docker.get_vm_info",
// 		trace.WithAttributes(attribute.String("vm_id", vmID)),
// 	)
// 	defer span.End()

// 	d.mutex.RLock()
// 	vm, exists := d.vmRegistry[vmID]
// 	d.mutex.RUnlock()

// 	if !exists {
// 		err := fmt.Errorf("vm %s not found", vmID)
// 		span.RecordError(err)
// 		return nil, err
// 	}

// 	info := &backendtypes.VMInfo{
// 		Config: vm.Config,
// 		State:  vm.State,
// 		// NetworkInfo: vm.NetworkInfo,
// 	}

// 	return info, nil
// }

// // GetVMMetrics returns metrics for a VM
// func (d *DockerBackend) GetVMMetrics(ctx context.Context, vmID string) (*backendtypes.VMMetrics, error) {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.get_vm_metrics",
// 		trace.WithAttributes(attribute.String("vm_id", vmID)),
// 	)
// 	defer span.End()

// 	d.mutex.RLock()
// 	vm, exists := d.vmRegistry[vmID]
// 	d.mutex.RUnlock()

// 	if !exists {
// 		err := fmt.Errorf("vm %s not found", vmID)
// 		span.RecordError(err)
// 		return nil, err
// 	}

// 	// Get container stats
// 	stats, err := d.dockerClient.ContainerStats(ctx, vm.ContainerID, false)
// 	if err != nil {
// 		span.RecordError(err)
// 		return nil, fmt.Errorf("failed to get container stats: %w", err)
// 	}
// 	defer stats.Body.Close()

// 	// Parse stats
// 	var dockerStats container.StatsResponse
// 	if err := json.NewDecoder(stats.Body).Decode(&dockerStats); err != nil {
// 		span.RecordError(err)
// 		return nil, fmt.Errorf("failed to decode container stats: %w", err)
// 	}

// 	// Convert to VM metrics
// 	metrics := &backendtypes.VMMetrics{
// 		Timestamp:        time.Now(),
// 		CpuTimeNanos:     safeUint64ToInt64(dockerStats.CPUStats.CPUUsage.TotalUsage),
// 		MemoryUsageBytes: safeUint64ToInt64(dockerStats.MemoryStats.Usage),
// 		DiskReadBytes:    0, // TODO: Calculate from BlkioStats
// 		DiskWriteBytes:   0, // TODO: Calculate from BlkioStats
// 		NetworkRxBytes:   0, // TODO: Calculate from NetworkStats
// 		NetworkTxBytes:   0, // TODO: Calculate from NetworkStats
// 	}

// 	// Calculate disk I/O
// 	for _, blkio := range dockerStats.BlkioStats.IoServiceBytesRecursive {
// 		if blkio.Op == "Read" {
// 			metrics.DiskReadBytes += safeUint64ToInt64(blkio.Value)
// 		} else if blkio.Op == "Write" {
// 			metrics.DiskWriteBytes += safeUint64ToInt64(blkio.Value)
// 		}
// 	}

// 	// Calculate network I/O
// 	if dockerStats.Networks != nil {
// 		for _, netStats := range dockerStats.Networks {
// 			metrics.NetworkRxBytes += safeUint64ToInt64(netStats.RxBytes)
// 			metrics.NetworkTxBytes += safeUint64ToInt64(netStats.TxBytes)
// 		}
// 	}

// 	return metrics, nil
// }

// // Ping checks if the Docker backend is healthy
// func (d *DockerBackend) Ping(ctx context.Context) error {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.ping")
// 	defer span.End()

// 	d.logger.DebugContext(ctx, "pinging Docker backend")

// 	// Ping Docker daemon
// 	if _, err := d.dockerClient.Ping(ctx); err != nil {
// 		span.RecordError(err)
// 		return fmt.Errorf("Docker daemon not available: %w", err)
// 	}

// 	return nil
// }

// // Helper methods

// // generateVMID generates a unique VM ID
// func (d *DockerBackend) generateVMID() string {
// 	return fmt.Sprintf("vm-%d", time.Now().UnixNano())
// }

// // vmConfigToContainerSpec converts VM configuration to container specification
// func (d *DockerBackend) vmConfigToContainerSpec(ctx context.Context, vmID string, config *metaldv1.VmConfig) (*ContainerSpec, error) {
// 	d.logger.DebugContext(ctx, "converting VM config to container spec", "vm_id", vmID)

// 	spec := &ContainerSpec{
// 		Labels: map[string]string{
// 			"unkey.vm.id":         vmID,
// 			"unkey.vm.created_by": "metald",
// 		},
// 		Memory: int64(config.GetMemorySizeMib()),
// 		CPUs:   float64(config.GetVcpuCount()),
// 	}

// 	// Docker image must be specified in metadata
// 	dockerImage, ok := config.GetMetadata()["docker_image"]
// 	if !ok || dockerImage == "" {
// 		return nil, fmt.Errorf("docker_image must be specified in VM config metadata")
// 	}
// 	spec.Image = dockerImage

// 	// Extract exposed ports from metadata
// 	if exposedPorts, ok := config.GetMetadata()["exposed_ports"]; ok {
// 		ports := strings.Split(exposedPorts, ",")
// 		for _, port := range ports {
// 			if port = strings.TrimSpace(port); port != "" {
// 				spec.ExposedPorts = append(spec.ExposedPorts, port)
// 			}
// 		}
// 	}

// 	// Extract environment variables from metadata
// 	if envVars, ok := config.GetMetadata()["env_vars"]; ok {
// 		vars := strings.Split(envVars, ",")
// 		for _, envVar := range vars {
// 			if envVar = strings.TrimSpace(envVar); envVar != "" {
// 				spec.Env = append(spec.Env, envVar)
// 			}
// 		}
// 	}

// 	// Allocate host ports for exposed ports
// 	for _, exposedPort := range spec.ExposedPorts {
// 		containerPort, err := strconv.Atoi(strings.Split(exposedPort, "/")[0])
// 		if err != nil {
// 			continue
// 		}

// 		protocol := "tcp"
// 		if strings.Contains(exposedPort, "/udp") {
// 			protocol = "udp"
// 		}

// 		// We'll allocate the port during container creation with retry logic
// 		spec.PortMappings = append(spec.PortMappings, PortMapping{
// 			ContainerPort: containerPort,
// 			HostPort:      0, // Will be allocated during creation
// 			Protocol:      protocol,
// 		})
// 	}

// 	return spec, nil
// }

// // createContainer creates a Docker container from the specification
// func (d *DockerBackend) createContainer(ctx context.Context, spec *ContainerSpec) (string, error) {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.create_container",
// 		trace.WithAttributes(attribute.String("image", spec.Image)),
// 	)
// 	defer span.End()

// 	d.logger.InfoContext(ctx, "checking if image exists locally", "image", spec.Image)
// 	_, err := d.dockerClient.ImageInspect(ctx, spec.Image)
// 	if err != nil {
// 		d.logger.InfoContext(ctx, "image not found locally, pulling image", "image", spec.Image, "error", err.Error())
// 		pullResponse, err := d.dockerClient.ImagePull(ctx, spec.Image, image.PullOptions{})
// 		if err != nil {
// 			return "", fmt.Errorf("failed to pull image %s: %w", spec.Image, err)
// 		}
// 		defer pullResponse.Close()

// 		// Read the pull response to completion to ensure pull finishes
// 		_, err = io.ReadAll(pullResponse)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to read pull response for image %s: %w", spec.Image, err)
// 		}

// 		d.logger.InfoContext(ctx, "image pulled successfully", "image", spec.Image)
// 	} else {
// 		d.logger.InfoContext(ctx, "image found locally, skipping pull", "image", spec.Image)
// 	}

// 	// Build container configuration
// 	config := &container.Config{
// 		Image:        spec.Image,
// 		Cmd:          spec.Cmd,
// 		Env:          spec.Env,
// 		Labels:       spec.Labels,
// 		ExposedPorts: make(nat.PortSet),
// 		WorkingDir:   spec.WorkingDir,
// 	}

// 	// Log the container command for debugging
// 	d.logger.InfoContext(ctx, "container configuration", "image", spec.Image, "cmd", config.Cmd, "env", config.Env)

// 	// Set up exposed ports
// 	for _, mapping := range spec.PortMappings {
// 		port := nat.Port(fmt.Sprintf("%d/%s", mapping.ContainerPort, mapping.Protocol))
// 		config.ExposedPorts[port] = struct{}{}
// 	}

// 	// Build host configuration
// 	hostConfig := &container.HostConfig{
// 		PortBindings: make(nat.PortMap),
// 		AutoRemove:   false, // Don't auto-remove containers for debugging
// 		Privileged:   d.config.Privileged,
// 		Resources: container.Resources{
// 			Memory:   spec.Memory,
// 			NanoCPUs: int64(spec.CPUs * 1e9),
// 		},
// 	}

// 	// Set up port bindings with retry logic
// 	maxRetries := 5
// 	for retry := 0; retry < maxRetries; retry++ {
// 		// Clear previous port bindings
// 		hostConfig.PortBindings = make(nat.PortMap)

// 		// Allocate ports for this attempt
// 		var allocatedPorts []int
// 		portAllocationFailed := false

// 		for i, mapping := range spec.PortMappings {
// 			if mapping.HostPort == 0 {
// 				// Allocate a new port
// 				hostPort, err := d.portAllocator.allocatePort(spec.Labels["unkey.vm.id"])
// 				if err != nil {
// 					// Release any ports allocated in this attempt
// 					for _, port := range allocatedPorts {
// 						d.portAllocator.releasePort(port, spec.Labels["unkey.vm.id"])
// 					}
// 					portAllocationFailed = true
// 					break
// 				}
// 				spec.PortMappings[i].HostPort = hostPort
// 				allocatedPorts = append(allocatedPorts, hostPort)
// 			}

// 			containerPort := nat.Port(fmt.Sprintf("%d/%s", mapping.ContainerPort, mapping.Protocol))
// 			hostConfig.PortBindings[containerPort] = []nat.PortBinding{
// 				{
// 					HostIP:   "0.0.0.0",
// 					HostPort: strconv.Itoa(spec.PortMappings[i].HostPort),
// 				},
// 			}
// 		}

// 		if portAllocationFailed {
// 			continue // Try again with new ports
// 		}

// 		// Create container
// 		containerName := d.config.ContainerPrefix + spec.Labels["unkey.vm.id"]
// 		resp, err := d.dockerClient.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
// 		if err != nil {
// 			// If it's a port binding error, release ports and try again
// 			if strings.Contains(err.Error(), "port is already allocated") || strings.Contains(err.Error(), "bind") {
// 				for _, port := range allocatedPorts {
// 					d.portAllocator.releasePort(port, spec.Labels["unkey.vm.id"])
// 				}
// 				d.logger.WarnContext(ctx, "port binding failed, retrying with new ports", "error", err, "retry", retry+1)
// 				continue
// 			}
// 			// Other errors are not retryable
// 			span.RecordError(err)
// 			return "", fmt.Errorf("failed to create container: %w", err)
// 		}

// 		// Success!
// 		span.SetAttributes(attribute.String("container_id", resp.ID))
// 		return resp.ID, nil
// 	}

// 	// If we get here, all retries failed
// 	return "", fmt.Errorf("failed to create container after %d retries due to port conflicts", maxRetries)
// }

// // getContainerNetworkInfo gets network information for a container
// func (d *DockerBackend) getContainerNetworkInfo(ctx context.Context, containerID string) (*metaldv1.VmNetworkInfo, error) {
// 	ctx, span := d.tracer.Start(ctx, "metald.docker.get_network_info",
// 		trace.WithAttributes(attribute.String("container_id", containerID)),
// 	)
// 	defer span.End()

// 	// Inspect container
// 	inspect, err := d.dockerClient.ContainerInspect(ctx, containerID)
// 	if err != nil {
// 		span.RecordError(err)
// 		return nil, fmt.Errorf("failed to inspect container: %w", err)
// 	}

// 	// Get network info from default network
// 	var networkInfo *metaldv1.VmNetworkInfo
// 	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Networks != nil {
// 		for networkName, network := range inspect.NetworkSettings.Networks {
// 			if network.IPAddress != "" {
// 				networkInfo = &metaldv1.VmNetworkInfo{
// 					IpAddress:  network.IPAddress,
// 					MacAddress: network.MacAddress,
// 					TapDevice:  networkName, // Use network name as tap device
// 				}
// 				break
// 			}
// 		}
// 	}

// 	// Add port mappings from container inspect
// 	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Ports != nil {
// 		var portMappings []*metaldv1.PortMapping
// 		for containerPort, hostBindings := range inspect.NetworkSettings.Ports {
// 			if len(hostBindings) > 0 {
// 				// Parse container port (e.g., "3000/tcp" -> 3000)
// 				portStr := strings.Split(string(containerPort), "/")[0]
// 				containerPortNum, err := strconv.Atoi(portStr)
// 				if err != nil {
// 					continue
// 				}

// 				// Get protocol (tcp/udp)
// 				protocol := "tcp"
// 				if strings.Contains(string(containerPort), "/udp") {
// 					protocol = "udp"
// 				}

// 				// Add mapping for each host binding
// 				for _, hostBinding := range hostBindings {
// 					hostPortNum, err := strconv.Atoi(hostBinding.HostPort)
// 					if err != nil {
// 						continue
// 					}

// 					portMappings = append(portMappings, &metaldv1.PortMapping{
// 						ContainerPort: safeIntToInt32(containerPortNum),
// 						HostPort:      safeIntToInt32(hostPortNum),
// 						Protocol:      protocol,
// 					})
// 				}
// 			}
// 		}

// 		// Initialize networkInfo if it doesn't exist
// 		if networkInfo == nil {
// 			networkInfo = &metaldv1.VmNetworkInfo{}
// 		}
// 		networkInfo.PortMappings = portMappings
// 	}

// 	return networkInfo, nil
// }

// // Port allocator methods

// // allocatePort allocates a port for a VM
// func (pa *portAllocator) allocatePort(vmID string) (int, error) {
// 	pa.mutex.Lock()
// 	defer pa.mutex.Unlock()

// 	// Try random ports to avoid conflicts using crypto/rand for security
// 	maxAttempts := 100
// 	for attempt := 0; attempt < maxAttempts; attempt++ {
// 		// Generate cryptographically secure random number
// 		portRange := int64(pa.maxPort - pa.minPort + 1)
// 		randomOffset, err := rand.Int(rand.Reader, big.NewInt(portRange))
// 		if err != nil {
// 			// Fallback to sequential allocation if crypto/rand fails
// 			for port := pa.minPort; port <= pa.maxPort; port++ {
// 				if _, exists := pa.allocated[port]; !exists {
// 					pa.allocated[port] = vmID
// 					return port, nil
// 				}
// 			}
// 			return 0, fmt.Errorf("failed to generate random port and no sequential ports available: %w", err)
// 		}

// 		port := int(randomOffset.Int64()) + pa.minPort
// 		if _, exists := pa.allocated[port]; !exists {
// 			pa.allocated[port] = vmID
// 			return port, nil
// 		}
// 	}

// 	return 0, fmt.Errorf("no available ports in range %d-%d after %d attempts", pa.minPort, pa.maxPort, maxAttempts)
// }

// // releasePort releases a port from a VM
// func (pa *portAllocator) releasePort(port int, vmID string) {
// 	pa.mutex.Lock()
// 	defer pa.mutex.Unlock()

// 	if allocated, exists := pa.allocated[port]; exists && allocated == vmID {
// 		delete(pa.allocated, port)
// 	}
// }

// // Ensure DockerBackend implements Backend interface
// var _ backendtypes.Backend = (*DockerBackend)(nil)
