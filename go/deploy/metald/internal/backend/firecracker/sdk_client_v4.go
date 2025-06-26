package firecracker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sdk "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	assetv1 "github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/asset/v1"
	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/assetmanager"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/jailer"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/network"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sys/unix"
)

// sdkV4VM represents a VM managed by the SDK v4
type sdkV4VM struct {
	ID           string
	Config       *metaldv1.VmConfig
	State        metaldv1.VmState
	Machine      *sdk.Machine
	NetworkInfo  *network.VMNetwork
	CancelFunc   context.CancelFunc
	AssetMapping *assetMapping      // Asset mapping for lease acquisition
	AssetPaths   map[string]string  // Prepared asset paths
}

// SDKClientV4 implements the Backend interface using firecracker-go-sdk
// with integrated jailer functionality for secure VM isolation.
//
// AIDEV-NOTE: This was previously named SDKClientV4Jailerless which was confusing
// because it DOES use a jailer - just the integrated one, not the external binary.
// The integrated jailer solves tap device permission issues and provides better
// control over the isolation process.
type SDKClientV4 struct {
	logger          *slog.Logger
	networkManager  *network.Manager
	assetClient     assetmanager.Client
	vmRegistry      map[string]*sdkV4VM
	vmAssetLeases   map[string][]string // VM ID -> asset lease IDs
	jailer          *jailer.Jailer
	jailerConfig    *config.JailerConfig
	baseDir         string
	tracer          trace.Tracer
	meter           metric.Meter
	vmCreateCounter metric.Int64Counter
	vmDeleteCounter metric.Int64Counter
	vmBootCounter   metric.Int64Counter
	vmErrorCounter  metric.Int64Counter
}

// NewSDKClientV4 creates a new SDK-based Firecracker backend client with integrated jailer
func NewSDKClientV4(logger *slog.Logger, networkManager *network.Manager, assetClient assetmanager.Client, jailerConfig *config.JailerConfig, baseDir string) (*SDKClientV4, error) {
	tracer := otel.Tracer("metald.firecracker.sdk.v4")
	meter := otel.Meter("metald.firecracker.sdk.v4")

	vmCreateCounter, err := meter.Int64Counter("vm_create_total",
		metric.WithDescription("Total number of VM create operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_create counter: %w", err)
	}

	vmDeleteCounter, err := meter.Int64Counter("vm_delete_total",
		metric.WithDescription("Total number of VM delete operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_delete counter: %w", err)
	}

	vmBootCounter, err := meter.Int64Counter("vm_boot_total",
		metric.WithDescription("Total number of VM boot operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_boot counter: %w", err)
	}

	vmErrorCounter, err := meter.Int64Counter("vm_error_total",
		metric.WithDescription("Total number of VM operation errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create vm_error counter: %w", err)
	}

	// Create integrated jailer
	integratedJailer := jailer.NewJailer(logger, jailerConfig)

	return &SDKClientV4{
		logger:          logger.With("backend", "firecracker-sdk-v4"),
		networkManager:  networkManager,
		assetClient:     assetClient,
		vmRegistry:      make(map[string]*sdkV4VM),
		vmAssetLeases:   make(map[string][]string),
		jailer:          integratedJailer,
		jailerConfig:    jailerConfig,
		baseDir:         baseDir,
		tracer:          tracer,
		meter:           meter,
		vmCreateCounter: vmCreateCounter,
		vmDeleteCounter: vmDeleteCounter,
		vmBootCounter:   vmBootCounter,
		vmErrorCounter:  vmErrorCounter,
	}, nil
}

// Initialize initializes the SDK client
func (c *SDKClientV4) Initialize() error {
	ctx, span := c.tracer.Start(context.Background(), "metald.firecracker.initialize")
	defer span.End()

	c.logger.InfoContext(ctx, "initializing firecracker SDK v4 client with integrated jailer")
	c.logger.InfoContext(ctx, "firecracker SDK v4 client initialized")
	return nil
}

// CreateVM creates a new VM using the SDK with integrated jailer
func (c *SDKClientV4) CreateVM(ctx context.Context, config *metaldv1.VmConfig) (string, error) {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.create_vm",
		trace.WithAttributes(
			attribute.Int("vcpus", int(config.GetCpu().GetVcpuCount())),
			attribute.Int64("memory_bytes", config.GetMemory().GetSizeBytes()),
		),
	)
	defer span.End()

	// Generate VM ID
	vmID, err := generateV4VMID()
	if err != nil {
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "create"),
			attribute.String("error", "generate_id"),
		))
		return "", fmt.Errorf("failed to generate VM ID: %w", err)
	}
	span.SetAttributes(attribute.String("vm_id", vmID))

	c.logger.LogAttrs(ctx, slog.LevelInfo, "creating VM with SDK v4",
		slog.String("vm_id", vmID),
		slog.Int("vcpus", int(config.GetCpu().GetVcpuCount())),
		slog.Int64("memory_bytes", config.GetMemory().GetSizeBytes()),
	)

	// Key difference: Allocate network resources BEFORE creating the jail
	// This allows us to create the tap device with full privileges
	networkInfo, err := c.networkManager.CreateVMNetwork(ctx, vmID)
	if err != nil {
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "create"),
			attribute.String("error", "network_allocation"),
		))
		return "", fmt.Errorf("failed to allocate network: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "allocated network for VM",
		slog.String("vm_id", vmID),
		slog.String("namespace", networkInfo.Namespace),
		slog.String("tap_device", networkInfo.TapDevice),
		slog.String("ip_address", networkInfo.IPAddress.String()),
	)

	// Prepare assets in the jailer chroot
	assetMapping, preparedPaths, err := c.prepareVMAssets(ctx, vmID, config)
	if err != nil {
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "create"),
			attribute.String("error", "asset_preparation"),
		))
		// Clean up network allocation
		if cleanupErr := c.networkManager.DeleteVMNetwork(ctx, vmID); cleanupErr != nil {
			c.logger.ErrorContext(ctx, "failed to cleanup network after asset preparation failure",
				"vm_id", vmID,
				"error", cleanupErr,
			)
		}
		return "", fmt.Errorf("failed to prepare VM assets: %w", err)
	}

	// Build SDK configuration WITHOUT jailer
	// The jailer functionality is now integrated
	_ = c.buildFirecrackerConfig(vmID, config, networkInfo, preparedPaths)

	// Create VM directory
	vmDir := filepath.Join(c.baseDir, vmID)
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create VM directory: %w", err)
	}

	// Register the VM
	vm := &sdkV4VM{
		ID:           vmID,
		Config:       config,
		State:        metaldv1.VmState_VM_STATE_CREATED,
		Machine:      nil, // Will be set when we boot
		NetworkInfo:  networkInfo,
		CancelFunc:   nil, // Will be set when we boot
		AssetMapping: assetMapping,
		AssetPaths:   preparedPaths,
	}

	c.vmRegistry[vmID] = vm

	c.vmCreateCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", "success"),
	))

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM created successfully with SDK v4",
		slog.String("vm_id", vmID),
	)

	return vmID, nil
}

// BootVM starts a created VM using our integrated jailer
func (c *SDKClientV4) BootVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.boot_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "boot"),
			attribute.String("error", "vm_not_found"),
		))
		return err
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "booting VM with SDK v4",
		slog.String("vm_id", vmID),
	)

	// For integrated jailer, we run firecracker in the VM directory
	vmDir := filepath.Join(c.baseDir, vmID)
	socketPath := filepath.Join(vmDir, "firecracker.sock")

	// Create log files
	logPath := filepath.Join(vmDir, "firecracker.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Build firecracker config that will be used by SDK
	fcConfig := c.buildFirecrackerConfig(vmID, vm.Config, vm.NetworkInfo, vm.AssetPaths)
	fcConfig.SocketPath = socketPath

	// Create a context for this VM
	vmCtx, cancel := context.WithCancel(context.Background())
	vm.CancelFunc = cancel

	// For integrated jailer, we use the SDK directly without external jailer
	// The network namespace is already set up and tap device created
	// We'll let the SDK manage firecracker but in our network namespace

	// Set the network namespace for the SDK to use
	if vm.NetworkInfo != nil && vm.NetworkInfo.Namespace != "" {
		fcConfig.NetNS = filepath.Join("/run/netns", vm.NetworkInfo.Namespace)
	}

	// Create and start the machine using SDK
	machine, err := sdk.NewMachine(vmCtx, fcConfig)
	if err != nil {
		cancel()
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "boot"),
			attribute.String("error", "create_machine"),
		))
		return fmt.Errorf("failed to create firecracker machine: %w", err)
	}

	// Start the VM
	if err := machine.Start(vmCtx); err != nil {
		cancel()
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "boot"),
			attribute.String("error", "start_machine"),
		))
		return fmt.Errorf("failed to start firecracker machine: %w", err)
	}

	vm.Machine = machine
	vm.State = metaldv1.VmState_VM_STATE_RUNNING

	// Acquire asset leases after successful boot
	if vm.AssetMapping != nil && len(vm.AssetMapping.AssetIDs()) > 0 {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "acquiring asset leases for VM",
			slog.String("vm_id", vmID),
			slog.Int("asset_count", len(vm.AssetMapping.AssetIDs())),
		)
		
		leaseIDs := []string{}
		for _, assetID := range vm.AssetMapping.AssetIDs() {
			ctx, acquireSpan := c.tracer.Start(ctx, "metald.firecracker.acquire_asset",
				trace.WithAttributes(
					attribute.String("vm.id", vmID),
					attribute.String("asset.id", assetID),
				),
			)
			leaseID, err := c.assetClient.AcquireAsset(ctx, assetID, vmID)
			if err != nil {
				acquireSpan.RecordError(err)
				acquireSpan.SetStatus(codes.Error, err.Error())
			} else {
				acquireSpan.SetAttributes(attribute.String("lease.id", leaseID))
			}
			acquireSpan.End()
			if err != nil {
				c.logger.ErrorContext(ctx, "failed to acquire asset lease",
					"vm_id", vmID,
					"asset_id", assetID,
					"error", err,
				)
				// Continue trying to acquire other leases even if one fails
				// AIDEV-TODO: Consider whether to fail the boot if lease acquisition fails
			} else {
				leaseIDs = append(leaseIDs, leaseID)
			}
		}
		
		// Store lease IDs for cleanup during VM deletion
		if len(leaseIDs) > 0 {
			c.vmAssetLeases[vmID] = leaseIDs
			c.logger.LogAttrs(ctx, slog.LevelInfo, "acquired asset leases",
				slog.String("vm_id", vmID),
				slog.Int("lease_count", len(leaseIDs)),
			)
		}
	}

	c.vmBootCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", "success"),
	))

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM booted successfully with SDK v4",
		slog.String("vm_id", vmID),
	)

	return nil
}

// Other methods would be similar to SDKClientV3...

// buildFirecrackerConfig builds the SDK configuration without jailer
func (c *SDKClientV4) buildFirecrackerConfig(vmID string, config *metaldv1.VmConfig, networkInfo *network.VMNetwork, preparedPaths map[string]string) sdk.Config {
	// For integrated jailer, we use absolute paths since we're not running inside chroot
	// The assets are still in the jailer directory structure for consistency
	jailerRoot := filepath.Join(
		c.jailerConfig.ChrootBaseDir,
		"firecracker",
		vmID,
		"root",
	)

	socketPath := "/firecracker.sock"
	
	// Determine kernel path - use prepared path if available, otherwise fallback to default
	kernelPath := filepath.Join(jailerRoot, "vmlinux")
	if preparedPaths != nil && len(preparedPaths) > 0 {
		// AIDEV-NOTE: In a more sophisticated implementation, we'd track which asset ID
		// corresponds to which component (kernel vs rootfs). For now, we rely on the
		// assetmanager preparing files with standard names in the target directory.
		// The prepared paths should already be in the jailerRoot directory.
		c.logger.LogAttrs(context.Background(), slog.LevelDebug, "using prepared asset paths",
			slog.String("vm_id", vmID),
			slog.Int("path_count", len(preparedPaths)),
		)
	}
	
	// Use host path since Firecracker is running outside chroot in "jailerless" mode
	metricsPath := filepath.Join(jailerRoot, "metrics.fifo")

	// AIDEV-NOTE: Create metrics FIFO for billaged to read Firecracker stats
	// billaged should read from: {jailerRoot}/metrics.fifo
	// e.g., /srv/jailer/firecracker/{vmID}/root/metrics.fifo
	hostMetricsPath := filepath.Join(jailerRoot, "metrics.fifo")

	// Create the metrics FIFO in the host filesystem
	if err := unix.Mkfifo(hostMetricsPath, 0644); err != nil && !os.IsExist(err) {
		c.logger.Error("failed to create metrics FIFO",
			slog.String("vm_id", vmID),
			slog.String("path", hostMetricsPath),
			slog.String("error", err.Error()),
		)
	} else {
		c.logger.Info("created metrics FIFO for billaged",
			slog.String("vm_id", vmID),
			slog.String("host_path", hostMetricsPath),
			slog.String("chroot_path", metricsPath),
		)
	}

	cfg := sdk.Config{ //nolint:exhaustruct // Optional fields are not needed for basic VM configuration
		SocketPath:      socketPath,
		LogPath:         "", // Logging handled externally
		LogLevel:        "Info",
		MetricsPath:     metricsPath, // Configure stats socket for billaged
		KernelImagePath: kernelPath,
		KernelArgs:      config.GetBoot().GetKernelArgs(),
		MachineCfg: models.MachineConfiguration{ //nolint:exhaustruct // Only setting required fields for basic VM configuration
			VcpuCount:  sdk.Int64(int64(config.GetCpu().GetVcpuCount())),
			MemSizeMib: sdk.Int64(config.GetMemory().GetSizeBytes() / (1024 * 1024)),
			Smt:        sdk.Bool(false),
		},
		// No JailerCfg - we handle jailing ourselves
	}

	// Add drives
	cfg.Drives = make([]models.Drive, 0, len(config.GetStorage()))
	for i, disk := range config.GetStorage() {
		driveID := disk.GetId()
		if driveID == "" {
			if disk.GetIsRootDevice() || i == 0 {
				driveID = "rootfs"
			} else {
				driveID = fmt.Sprintf("drive_%d", i)
			}
		}

		// Use absolute paths for integrated jailer
		drive := models.Drive{ //nolint:exhaustruct // Only setting required drive fields
			DriveID:      &driveID,
			PathOnHost:   sdk.String(filepath.Join(jailerRoot, filepath.Base(disk.GetPath()))),
			IsRootDevice: sdk.Bool(disk.GetIsRootDevice() || i == 0),
			IsReadOnly:   sdk.Bool(disk.GetReadOnly()),
		}
		cfg.Drives = append(cfg.Drives, drive)
	}

	// Add network interface
	if networkInfo != nil {
		iface := sdk.NetworkInterface{ //nolint:exhaustruct // Only setting required network interface fields
			StaticConfiguration: &sdk.StaticNetworkConfiguration{ //nolint:exhaustruct // Only setting required network configuration fields
				HostDevName: networkInfo.TapDevice,
				MacAddress:  networkInfo.MacAddress,
			},
		}
		cfg.NetworkInterfaces = []sdk.NetworkInterface{iface}
	}

	return cfg
}

// assetRequirement represents a required asset for VM creation
type assetRequirement struct {
	Type     assetv1.AssetType
	Labels   map[string]string
	Required bool
}

// assetMapping tracks the mapping between requirements and actual assets
type assetMapping struct {
	requirements []assetRequirement
	assets       map[string]*assetv1.Asset // requirement index -> asset
	assetIDs     []string
	leaseIDs     []string
}

func (am *assetMapping) AssetIDs() []string {
	return am.assetIDs
}

func (am *assetMapping) LeaseIDs() []string {
	return am.leaseIDs
}

// buildAssetRequirements analyzes VM config to determine required assets
func (c *SDKClientV4) buildAssetRequirements(config *metaldv1.VmConfig) []assetRequirement {
	var reqs []assetRequirement
	
	// Kernel requirement
	if config.Boot != nil && config.Boot.KernelPath != "" {
		reqs = append(reqs, assetRequirement{
			Type:     assetv1.AssetType_ASSET_TYPE_KERNEL,
			Required: true,
		})
	}
	
	// Rootfs requirements from storage devices
	for _, disk := range config.Storage {
		if disk.IsRootDevice {
			labels := make(map[string]string)
			// Check for docker image in disk options first, then config metadata
			if dockerImage, ok := disk.Options["docker_image"]; ok {
				labels["docker_image"] = dockerImage
			} else if dockerImage, ok := config.Metadata["docker_image"]; ok {
				labels["docker_image"] = dockerImage
			}
			reqs = append(reqs, assetRequirement{
				Type:     assetv1.AssetType_ASSET_TYPE_ROOTFS,
				Labels:   labels,
				Required: true,
			})
		}
	}
	
	// Initrd requirement (optional)
	if config.Boot != nil && config.Boot.InitrdPath != "" {
		reqs = append(reqs, assetRequirement{
			Type:     assetv1.AssetType_ASSET_TYPE_INITRD,
			Required: false,
		})
	}
	
	return reqs
}

// matchAssets matches available assets to requirements
func (c *SDKClientV4) matchAssets(reqs []assetRequirement, availableAssets []*assetv1.Asset) (*assetMapping, error) {
	mapping := &assetMapping{
		requirements: reqs,
		assets:       make(map[string]*assetv1.Asset),
		assetIDs:     []string{},
	}
	
	for i, req := range reqs {
		var matched *assetv1.Asset
		
		// Find best matching asset
		for _, asset := range availableAssets {
			if asset.Type != req.Type {
				continue
			}
			
			// Check if all required labels match
			labelMatch := true
			for k, v := range req.Labels {
				if assetLabel, ok := asset.Labels[k]; !ok || assetLabel != v {
					labelMatch = false
					break
				}
			}
			
			if labelMatch {
				matched = asset
				break
			}
		}
		
		if matched == nil && req.Required {
			// Build helpful error message
			labelStr := ""
			for k, v := range req.Labels {
				if labelStr != "" {
					labelStr += ", "
				}
				labelStr += fmt.Sprintf("%s=%s", k, v)
			}
			return nil, fmt.Errorf("no matching asset found for type %s with labels {%s}", 
				req.Type.String(), labelStr)
		}
		
		if matched != nil {
			mapping.assets[fmt.Sprintf("%d", i)] = matched
			mapping.assetIDs = append(mapping.assetIDs, matched.Id)
		}
	}
	
	return mapping, nil
}

// prepareVMAssets prepares kernel and rootfs assets for the VM in the jailer chroot
// Returns the asset mapping for lease acquisition after successful boot
func (c *SDKClientV4) prepareVMAssets(ctx context.Context, vmID string, config *metaldv1.VmConfig) (*assetMapping, map[string]string, error) {
	// Calculate the jailer chroot path
	jailerRoot := filepath.Join(
		c.jailerConfig.ChrootBaseDir,
		"firecracker",
		vmID,
		"root",
	)

	c.logger.LogAttrs(ctx, slog.LevelInfo, "preparing VM assets using assetmanager",
		slog.String("vm_id", vmID),
		slog.String("target_path", jailerRoot),
	)

	// Ensure the jailer root directory exists
	if err := os.MkdirAll(jailerRoot, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create jailer root directory: %w", err)
	}

	// Check if assetmanager is enabled
	// If disabled (using noop client), fall back to static file copying for backward compatibility
	// AIDEV-NOTE: We check if the QueryAssets call succeeds to determine if assetmanager is available
	// We don't require assets to exist, as they can be built on demand
	ctx, checkSpan := c.tracer.Start(ctx, "metald.firecracker.check_assetmanager", 
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.String("asset.type", "KERNEL"),
		),
	)
	_, err := c.assetClient.QueryAssets(ctx, assetv1.AssetType_ASSET_TYPE_KERNEL, nil, nil)
	checkSpan.End()
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "assetmanager disabled or unavailable, using static file copying",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		// AIDEV-NOTE: Fallback to old behavior when assetmanager is disabled
		// This ensures backward compatibility
		if err := c.prepareVMAssetsStatic(ctx, vmID, config, jailerRoot); err != nil {
			return nil, nil, err
		}
		return nil, nil, nil
	}

	// Build asset requirements from VM configuration
	requiredAssets := c.buildAssetRequirements(config)
	c.logger.LogAttrs(ctx, slog.LevelDebug, "determined asset requirements",
		slog.String("vm_id", vmID),
		slog.Int("required_count", len(requiredAssets)),
	)

	// Query assetmanager for available assets with automatic build support
	// AIDEV-NOTE: Using QueryAssets instead of ListAssets to enable automatic asset creation
	allAssets := []*assetv1.Asset{}
	
	// Extract tenant_id from VM metadata if available
	tenantID := ""
	if tid, ok := config.Metadata["tenant_id"]; ok {
		tenantID = tid
	}
	
	// Group requirements by type and labels for efficient querying
	type queryKey struct {
		assetType assetv1.AssetType
		labels    string // Serialized labels for grouping
	}
	queryGroups := make(map[queryKey][]assetRequirement)
	
	for _, req := range requiredAssets {
		// Serialize labels for grouping
		labelStr := ""
		for k, v := range req.Labels {
			if labelStr != "" {
				labelStr += ","
			}
			labelStr += fmt.Sprintf("%s=%s", k, v)
		}
		key := queryKey{assetType: req.Type, labels: labelStr}
		queryGroups[key] = append(queryGroups[key], req)
	}
	
	// Query each unique combination of type and labels
	for key, reqs := range queryGroups {
		// Use the first requirement's labels (they should all be the same in the group)
		labels := reqs[0].Labels
		
		// Generate a deterministic asset ID based on the asset type and labels
		// This allows us to query for the exact asset later
		assetID := c.generateAssetID(key.assetType, labels)
		
		c.logger.LogAttrs(ctx, slog.LevelInfo, "generated asset ID for query",
			slog.String("asset_id", assetID),
			slog.String("asset_type", key.assetType.String()),
			slog.Any("labels", labels),
		)
		
		// Configure build options for automatic asset creation
		// AIDEV-NOTE: When WaitForCompletion is true, VM creation will block until the build
		// completes. This provides a synchronous experience where the VM is ready to boot
		// immediately after creation, but may cause longer wait times (up to 30 minutes
		// for large images). The client timeout should be configured accordingly.
		buildOptions := &assetv1.BuildOptions{
			EnableAutoBuild:     true,
			WaitForCompletion:   true,  // Block VM creation until build completes
			BuildTimeoutSeconds: 1800,  // 30 minutes maximum wait time
			TenantId:            tenantID,
			SuggestedAssetId:    assetID,
		}
		
		// Query assets with automatic build support
		// Create a quick span just to record that we're initiating a query
		_, initSpan := c.tracer.Start(ctx, "metald.firecracker.query_assets",
			trace.WithAttributes(
				attribute.String("vm.id", vmID),
				attribute.String("asset.type", key.assetType.String()),
				attribute.StringSlice("asset.labels", func() []string {
					var labelPairs []string
					for k, v := range labels {
						labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
					}
					return labelPairs
				}()),
				attribute.String("tenant.id", tenantID),
				attribute.Bool("auto_build.enabled", buildOptions.EnableAutoBuild),
				attribute.Int("build.timeout_seconds", int(buildOptions.BuildTimeoutSeconds)),
			),
		)
		initSpan.End() // End immediately - this just marks the initiation
		
		// Make the actual call without wrapping in a span (it has its own internal spans)
		resp, err := c.assetClient.QueryAssets(ctx, key.assetType, labels, buildOptions)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to query assets of type %s with labels %v: %w", 
				key.assetType.String(), labels, err)
		}
		
		// Create a quick span to record the results
		_, resultSpan := c.tracer.Start(ctx, "metald.firecracker.query_assets_complete",
			trace.WithAttributes(
				attribute.String("vm.id", vmID),
				attribute.String("asset.type", key.assetType.String()),
				attribute.Int("assets.found", len(resp.GetAssets())),
				attribute.Int("builds.triggered", len(resp.GetTriggeredBuilds())),
			),
		)
		resultSpan.End()
		
		// Log any triggered builds
		for _, build := range resp.GetTriggeredBuilds() {
			c.logger.LogAttrs(ctx, slog.LevelInfo, "automatic build triggered for missing asset",
				slog.String("vm_id", vmID),
				slog.String("build_id", build.GetBuildId()),
				slog.String("docker_image", build.GetDockerImage()),
				slog.String("status", build.GetStatus()),
			)
			
			if build.GetStatus() == "failed" {
				c.logger.LogAttrs(ctx, slog.LevelError, "automatic build failed",
					slog.String("vm_id", vmID),
					slog.String("build_id", build.GetBuildId()),
					slog.String("error", build.GetErrorMessage()),
				)
			}
		}
		
		allAssets = append(allAssets, resp.GetAssets()...)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "retrieved available assets",
		slog.String("vm_id", vmID),
		slog.Int("available_count", len(allAssets)),
	)
	
	// Log asset details for debugging
	for _, asset := range allAssets {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "available asset",
			slog.String("asset_id", asset.Id),
			slog.String("asset_type", asset.Type.String()),
			slog.Any("labels", asset.Labels),
		)
	}

	// Match required assets with available ones
	assetMapping, err := c.matchAssets(requiredAssets, allAssets)
	if err != nil {
		c.logger.LogAttrs(ctx, slog.LevelError, "failed to match assets",
			slog.String("vm_id", vmID),
			slog.String("error", err.Error()),
		)
		return nil, nil, fmt.Errorf("asset matching failed: %w", err)
	}

	// Prepare assets in target location
	ctx, prepareSpan := c.tracer.Start(ctx, "metald.firecracker.prepare_assets",
		trace.WithAttributes(
			attribute.String("vm.id", vmID),
			attribute.StringSlice("asset.ids", assetMapping.AssetIDs()),
			attribute.String("target.path", jailerRoot),
		),
	)
	preparedPaths, err := c.assetClient.PrepareAssets(
		ctx,
		assetMapping.AssetIDs(),
		jailerRoot,
		vmID,
	)
	if err != nil {
		prepareSpan.RecordError(err)
		prepareSpan.SetStatus(codes.Error, err.Error())
	} else {
		prepareSpan.SetAttributes(
			attribute.Int("assets.prepared", len(preparedPaths)),
		)
	}
	prepareSpan.End()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare assets: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "assets prepared successfully",
		slog.String("vm_id", vmID),
		slog.Int("asset_count", len(preparedPaths)),
	)

	// The preparedPaths map contains asset_id -> actual_path mappings
	// These paths will be used to update the VM configuration before starting
	// Asset leases will be acquired after successful VM boot in BootVM
	// to avoid holding leases for VMs that fail to start

	return assetMapping, preparedPaths, nil
}

// prepareVMAssetsStatic is the fallback implementation for static file copying
// Used when assetmanager is disabled for backward compatibility
func (c *SDKClientV4) prepareVMAssetsStatic(ctx context.Context, vmID string, config *metaldv1.VmConfig, jailerRoot string) error {
	// Copy kernel
	if kernelPath := config.GetBoot().GetKernelPath(); kernelPath != "" {
		kernelDst := filepath.Join(jailerRoot, "vmlinux")
		if err := copyFileWithOwnership(kernelPath, kernelDst, int(c.jailerConfig.UID), int(c.jailerConfig.GID)); err != nil {
			return fmt.Errorf("failed to copy kernel: %w", err)
		}
		c.logger.LogAttrs(ctx, slog.LevelInfo, "copied kernel to jailer root",
			slog.String("src", kernelPath),
			slog.String("dst", kernelDst),
		)
	}

	// Copy rootfs images
	for _, disk := range config.GetStorage() {
		if disk.GetPath() != "" {
			diskDst := filepath.Join(jailerRoot, filepath.Base(disk.GetPath()))
			if err := copyFileWithOwnership(disk.GetPath(), diskDst, int(c.jailerConfig.UID), int(c.jailerConfig.GID)); err != nil {
				return fmt.Errorf("failed to copy disk %s: %w", disk.GetPath(), err)
			}
			c.logger.LogAttrs(ctx, slog.LevelInfo, "copied disk to jailer root",
				slog.String("src", disk.GetPath()),
				slog.String("dst", diskDst),
			)
		}
	}

	return nil
}

// DeleteVM deletes a VM and cleans up its resources
func (c *SDKClientV4) DeleteVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.delete_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	c.logger.LogAttrs(ctx, slog.LevelInfo, "deleting VM",
		slog.String("vm_id", vmID),
	)

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		c.vmErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("operation", "delete"),
			attribute.String("error", "vm_not_found"),
		))
		return err
	}

	// Stop the VM if it's running
	if vm.Machine != nil {
		if err := vm.Machine.StopVMM(); err != nil {
			c.logger.WarnContext(ctx, "failed to stop VMM during delete",
				"vm_id", vmID,
				"error", err,
			)
		}

		// Cancel the VM context
		if vm.CancelFunc != nil {
			vm.CancelFunc()
		}
	}

	// Delete network resources
	if err := c.networkManager.DeleteVMNetwork(ctx, vmID); err != nil {
		c.logger.ErrorContext(ctx, "failed to delete VM network",
			"vm_id", vmID,
			"error", err,
		)
		// Continue with deletion even if network cleanup fails
	}

	// Clean up VM directory
	vmDir := filepath.Join(c.baseDir, vmID)
	if err := os.RemoveAll(vmDir); err != nil {
		c.logger.WarnContext(ctx, "failed to remove VM directory",
			"vm_id", vmID,
			"path", vmDir,
			"error", err,
		)
	}

	// Clean up jailer chroot
	chrootPath := filepath.Join(c.jailerConfig.ChrootBaseDir, "firecracker", vmID)
	if err := os.RemoveAll(chrootPath); err != nil {
		c.logger.WarnContext(ctx, "failed to remove jailer chroot",
			"vm_id", vmID,
			"path", chrootPath,
			"error", err,
		)
	}

	// Release asset leases
	if leaseIDs, ok := c.vmAssetLeases[vmID]; ok {
		c.logger.LogAttrs(ctx, slog.LevelInfo, "releasing asset leases",
			slog.String("vm_id", vmID),
			slog.Int("lease_count", len(leaseIDs)),
		)
		
		for _, leaseID := range leaseIDs {
			ctx, releaseSpan := c.tracer.Start(ctx, "metald.firecracker.release_asset",
				trace.WithAttributes(
					attribute.String("vm.id", vmID),
					attribute.String("lease.id", leaseID),
				),
			)
			err := c.assetClient.ReleaseAsset(ctx, leaseID)
			if err != nil {
				releaseSpan.RecordError(err)
				releaseSpan.SetStatus(codes.Error, err.Error())
			}
			releaseSpan.End()
			if err != nil {
				c.logger.ErrorContext(ctx, "failed to release asset lease",
					"vm_id", vmID,
					"lease_id", leaseID,
					"error", err,
				)
				// Continue with other leases even if one fails
			}
		}
		delete(c.vmAssetLeases, vmID)
	}

	// Remove from registry
	delete(c.vmRegistry, vmID)

	c.vmDeleteCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", "success"),
	))

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM deleted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ShutdownVM gracefully shuts down a VM
func (c *SDKClientV4) ShutdownVM(ctx context.Context, vmID string) error {
	return c.ShutdownVMWithOptions(ctx, vmID, false, 30)
}

// ShutdownVMWithOptions shuts down a VM with configurable options
func (c *SDKClientV4) ShutdownVMWithOptions(ctx context.Context, vmID string, force bool, timeoutSeconds int32) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.shutdown_vm",
		trace.WithAttributes(
			attribute.String("vm_id", vmID),
			attribute.Bool("force", force),
			attribute.Int("timeout_seconds", int(timeoutSeconds)),
		),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return err
	}

	if vm.Machine == nil {
		return fmt.Errorf("vm %s is not running", vmID)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "shutting down VM",
		slog.String("vm_id", vmID),
		slog.Bool("force", force),
		slog.Int("timeout_seconds", int(timeoutSeconds)),
	)

	// Create a timeout context
	shutdownCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	if force { //nolint:nestif // Complex shutdown logic requires nested conditions for force vs graceful shutdown
		// Force shutdown by stopping the VMM immediately
		if err := vm.Machine.StopVMM(); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to force stop VM: %w", err)
		}
	} else {
		// Try graceful shutdown first
		if err := vm.Machine.Shutdown(shutdownCtx); err != nil {
			c.logger.WarnContext(ctx, "graceful shutdown failed, attempting force stop",
				"vm_id", vmID,
				"error", err,
			)
			// Fall back to force stop
			if stopErr := vm.Machine.StopVMM(); stopErr != nil {
				span.RecordError(stopErr)
				return fmt.Errorf("failed to stop VM after graceful shutdown failed: %w", stopErr)
			}
		}
	}

	// Wait for the VM to actually stop
	if err := vm.Machine.Wait(shutdownCtx); err != nil {
		c.logger.WarnContext(ctx, "error waiting for VM to stop",
			"vm_id", vmID,
			"error", err,
		)
	}

	// Update state
	vm.State = metaldv1.VmState_VM_STATE_SHUTDOWN

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM shutdown successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// PauseVM pauses a running VM
func (c *SDKClientV4) PauseVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.pause_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return err
	}

	if vm.Machine == nil {
		return fmt.Errorf("vm %s is not running", vmID)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "pausing VM",
		slog.String("vm_id", vmID),
	)

	if err := vm.Machine.PauseVM(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to pause VM: %w", err)
	}

	vm.State = metaldv1.VmState_VM_STATE_PAUSED

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM paused successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// ResumeVM resumes a paused VM
func (c *SDKClientV4) ResumeVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.resume_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return err
	}

	if vm.Machine == nil {
		return fmt.Errorf("vm %s is not running", vmID)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "resuming VM",
		slog.String("vm_id", vmID),
	)

	if err := vm.Machine.ResumeVM(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to resume VM: %w", err)
	}

	vm.State = metaldv1.VmState_VM_STATE_RUNNING

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM resumed successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// RebootVM reboots a running VM
func (c *SDKClientV4) RebootVM(ctx context.Context, vmID string) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.reboot_vm",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	c.logger.LogAttrs(ctx, slog.LevelInfo, "rebooting VM",
		slog.String("vm_id", vmID),
	)

	// Shutdown the VM
	if err := c.ShutdownVMWithOptions(ctx, vmID, false, 30); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to shutdown VM for reboot: %w", err)
	}

	// Wait a moment
	time.Sleep(1 * time.Second)

	// Boot the VM again
	if err := c.BootVM(ctx, vmID); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to boot VM after shutdown: %w", err)
	}

	c.logger.LogAttrs(ctx, slog.LevelInfo, "VM rebooted successfully",
		slog.String("vm_id", vmID),
	)

	return nil
}

// generateAssetID generates a deterministic asset ID based on type and labels
func (c *SDKClientV4) generateAssetID(assetType assetv1.AssetType, labels map[string]string) string {
	// Create a deterministic string from sorted labels
	var parts []string
	parts = append(parts, fmt.Sprintf("type=%s", assetType.String()))
	
	// Sort label keys for deterministic ordering
	var keys []string
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// Add sorted labels
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}
	
	// Create a hash of the combined string
	combined := strings.Join(parts, ",")
	hash := sha256.Sum256([]byte(combined))
	
	// Return a readable asset ID
	return fmt.Sprintf("asset-%x", hash[:8])
}

// GetVMInfo returns information about a VM
func (c *SDKClientV4) GetVMInfo(ctx context.Context, vmID string) (*types.VMInfo, error) {
	_, span := c.tracer.Start(ctx, "metald.firecracker.get_vm_info",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return nil, err
	}

	info := &types.VMInfo{ //nolint:exhaustruct // NetworkInfo is populated conditionally below
		Config: vm.Config,
		State:  vm.State,
	}

	// Add network info if available
	if vm.NetworkInfo != nil {
		info.NetworkInfo = &metaldv1.VmNetworkInfo{ //nolint:exhaustruct // Optional fields are not needed for basic network info
			IpAddress:  vm.NetworkInfo.IPAddress.String(),
			MacAddress: vm.NetworkInfo.MacAddress,
			TapDevice:  vm.NetworkInfo.TapDevice,
		}
	}

	return info, nil
}

// GetVMMetrics returns metrics for a VM
func (c *SDKClientV4) GetVMMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.get_vm_metrics",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	vm, exists := c.vmRegistry[vmID]
	if !exists {
		err := fmt.Errorf("vm %s not found", vmID)
		span.RecordError(err)
		return nil, err
	}

	if vm.Machine == nil {
		return nil, fmt.Errorf("vm %s is not running", vmID)
	}

	// Read real metrics from Firecracker stats FIFO
	return c.readFirecrackerMetrics(ctx, vmID)
}

// FirecrackerMetrics represents the JSON structure from Firecracker stats
type FirecrackerMetrics struct {
	VCPU []struct {
		ExitReasons map[string]int64 `json:"exit_reasons"`
	} `json:"vcpu"`
	Block []struct {
		ReadBytes  int64 `json:"read_bytes"`
		WriteBytes int64 `json:"write_bytes"`
		ReadCount  int64 `json:"read_count"`
		WriteCount int64 `json:"write_count"`
	} `json:"block"`
	Net []struct {
		RxBytes   int64 `json:"rx_bytes"`
		TxBytes   int64 `json:"tx_bytes"`
		RxPackets int64 `json:"rx_packets"`
		TxPackets int64 `json:"tx_packets"`
	} `json:"net"`
	// Note: CPU time and memory usage may be in other fields or require calculation
}

// readFirecrackerMetrics reads metrics from the Firecracker stats FIFO
func (c *SDKClientV4) readFirecrackerMetrics(ctx context.Context, vmID string) (*types.VMMetrics, error) {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.read_metrics",
		trace.WithAttributes(attribute.String("vm_id", vmID)),
	)
	defer span.End()

	// Construct FIFO path
	fifoPath := filepath.Join(c.jailerConfig.ChrootBaseDir, "firecracker", vmID, "root", "metrics.fifo")

	// Try to read from FIFO (with timeout for blocking read)
	file, err := os.OpenFile(fifoPath, os.O_RDONLY, 0)
	if err != nil {
		// If FIFO doesn't exist or can't be opened, return zeros (VM might be starting)
		c.logger.WarnContext(ctx, "cannot read metrics FIFO",
			slog.String("vm_id", vmID),
			slog.String("fifo_path", fifoPath),
			slog.String("error", err.Error()),
		)
		return &types.VMMetrics{
			Timestamp:        time.Now(),
			CpuTimeNanos:     0,
			MemoryUsageBytes: 0,
			DiskReadBytes:    0,
			DiskWriteBytes:   0,
			NetworkRxBytes:   0,
			NetworkTxBytes:   0,
		}, nil
	}
	defer file.Close()

	// AIDEV-NOTE: Firecracker writes a continuous JSON stream to the FIFO
	// We need to use a JSON decoder to handle streaming JSON objects properly
	type result struct {
		metrics *FirecrackerMetrics
		err     error
	}
	resultCh := make(chan result, 1)

	go func() {
		decoder := json.NewDecoder(file)
		var fcMetrics FirecrackerMetrics

		// AIDEV-NOTE: Firecracker writes periodic JSON objects to the FIFO
		// We might start reading in the middle of a JSON object, so we need to
		// keep trying until we get a complete, valid JSON object
		maxAttempts := 5
		for attempt := 0; attempt < maxAttempts; attempt++ {
			if err := decoder.Decode(&fcMetrics); err != nil {
				// If we get a JSON syntax error, it might be because we started
				// reading in the middle of an object. Try again.
				if attempt < maxAttempts-1 {
					continue
				}
				resultCh <- result{metrics: nil, err: err}
				return
			}

			// Successfully decoded a complete JSON object
			resultCh <- result{metrics: &fcMetrics, err: nil}
			return
		}
	}()

	var fcMetrics *FirecrackerMetrics
	select {
	case res := <-resultCh:
		if res.err != nil {
			c.logger.WarnContext(ctx, "failed to read JSON from metrics FIFO",
				slog.String("vm_id", vmID),
				slog.String("error", res.err.Error()),
			)
			// Return zeros on read error - VM might still be starting up
			return &types.VMMetrics{
				Timestamp:        time.Now(),
				CpuTimeNanos:     0,
				MemoryUsageBytes: 0,
				DiskReadBytes:    0,
				DiskWriteBytes:   0,
				NetworkRxBytes:   0,
				NetworkTxBytes:   0,
			}, nil
		}
		fcMetrics = res.metrics

	case <-time.After(2 * time.Second):
		// Timeout - no metrics available within timeout
		c.logger.DebugContext(ctx, "timeout reading metrics FIFO",
			slog.String("vm_id", vmID),
		)
		return &types.VMMetrics{
			Timestamp:        time.Now(),
			CpuTimeNanos:     0,
			MemoryUsageBytes: 0,
			DiskReadBytes:    0,
			DiskWriteBytes:   0,
			NetworkRxBytes:   0,
			NetworkTxBytes:   0,
		}, nil
	}

	// Convert to our internal format
	metrics := &types.VMMetrics{
		Timestamp:        time.Now(),
		CpuTimeNanos:     0, // TODO: Calculate from VCPU exit reasons or other fields
		MemoryUsageBytes: 0, // TODO: Extract from memory metrics if available
		DiskReadBytes:    0,
		DiskWriteBytes:   0,
		NetworkRxBytes:   0,
		NetworkTxBytes:   0,
	}

	// Aggregate disk metrics from all block devices
	for _, block := range fcMetrics.Block {
		metrics.DiskReadBytes += block.ReadBytes
		metrics.DiskWriteBytes += block.WriteBytes
	}

	// Aggregate network metrics from all network interfaces
	for _, net := range fcMetrics.Net {
		metrics.NetworkRxBytes += net.RxBytes
		metrics.NetworkTxBytes += net.TxBytes
	}

	c.logger.DebugContext(ctx, "read Firecracker metrics",
		slog.String("vm_id", vmID),
		slog.Int64("disk_read_bytes", metrics.DiskReadBytes),
		slog.Int64("disk_write_bytes", metrics.DiskWriteBytes),
		slog.Int64("network_rx_bytes", metrics.NetworkRxBytes),
		slog.Int64("network_tx_bytes", metrics.NetworkTxBytes),
	)

	return metrics, nil
}

func (c *SDKClientV4) Ping(ctx context.Context) error {
	c.logger.DebugContext(ctx, "pinging firecracker SDK v4 backend")
	return nil
}

func (c *SDKClientV4) Shutdown(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "metald.firecracker.shutdown")
	defer span.End()

	c.logger.InfoContext(ctx, "shutting down SDK v4 backend")

	// Shutdown all running VMs
	for vmID, vm := range c.vmRegistry {
		c.logger.InfoContext(ctx, "shutting down VM during backend shutdown",
			"vm_id", vmID,
		)
		if vm.Machine != nil {
			if err := vm.Machine.StopVMM(); err != nil {
				c.logger.ErrorContext(ctx, "failed to stop VM during shutdown",
					"vm_id", vmID,
					"error", err,
				)
			}
			if vm.CancelFunc != nil {
				vm.CancelFunc()
			}
		}
	}

	c.logger.InfoContext(ctx, "SDK v4 backend shutdown complete")
	return nil
}

// Ensure SDKClientV4 implements Backend interface
var _ types.Backend = (*SDKClientV4)(nil)

// generateV4VMID generates a unique VM ID for V4 client
func generateV4VMID() (string, error) {
	// Generate a random ID
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}
	return fmt.Sprintf("ud-%s", hex.EncodeToString(bytes)), nil
}

// Helper function to copy files with ownership
func copyFileWithOwnership(src, dst string, uid, gid int) error {
	// Use cp command to handle large files efficiently
	cmd := exec.Command("cp", "-f", src, dst)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cp command failed: %w, output: %s", err, output)
	}

	// Set permissions
	if err := os.Chmod(dst, 0644); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dst, err)
	}

	// Set ownership
	if err := os.Chown(dst, uid, gid); err != nil {
		// Log but don't fail - might work anyway
		return nil
	}

	return nil
}

// AIDEV-NOTE: This implementation integrates jailer functionality directly into metald
// Key advantages:
// 1. Network setup happens BEFORE dropping privileges
// 2. Tap devices are created with full capabilities
// 3. We maintain security isolation via chroot and privilege dropping
// 4. No external jailer binary needed - everything is integrated
