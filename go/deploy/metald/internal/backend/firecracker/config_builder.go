package firecracker

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	sdk "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/network"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"golang.org/x/sys/unix"
)

// buildFirecrackerConfig builds the SDK configuration without jailer
func (c *Client) buildFirecrackerConfig(ctx context.Context, vmID string, config *metaldv1.VmConfig, networkInfo *network.VMNetwork, preparedPaths map[string]string) sdk.Config {
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
	if len(preparedPaths) > 0 {
		// In a more sophisticated implementation, we'd track which asset ID
		// corresponds to which component (kernel vs rootfs). For now, we rely on the
		// assetmanager preparing files with standard names in the target directory.
		c.logger.LogAttrs(ctx, slog.LevelDebug, "using prepared asset paths",
			slog.String("vm_id", vmID),
			slog.Int("path_count", len(preparedPaths)),
		)
	}

	// Setup metrics FIFO for billaged
	metricsPath := c.setupMetricsFIFO(ctx, vmID, jailerRoot)

	// Setup console logging
	consoleLogPath := filepath.Join(jailerRoot, "console.log")
	consoleFifoPath := filepath.Join(jailerRoot, "console.fifo")

	// Use the kernel args as provided by the caller
	// Metadata handling is now done in BootVM
	kernelArgs := config.GetBoot()

	// Build the configuration
	cfg := c.buildSDKConfig(
		socketPath,
		consoleLogPath,
		consoleFifoPath,
		metricsPath,
		kernelPath,
		kernelArgs,
		config,
		jailerRoot,
	)

	// Add network interface
	if networkInfo != nil {
		c.addNetworkInterfaceToConfig(&cfg, networkInfo)
	}

	return cfg
}

// setupMetricsFIFO creates the metrics FIFO for billaged to read Firecracker stats
func (c *Client) setupMetricsFIFO(ctx context.Context, vmID string, jailerRoot string) string {
	metricsPath := filepath.Join(jailerRoot, "metrics.fifo")
	hostMetricsPath := filepath.Join(jailerRoot, "metrics.fifo")

	// Create the metrics FIFO in the host filesystem
	if err := unix.Mkfifo(hostMetricsPath, 0o644); err != nil && !os.IsExist(err) {
		c.logger.ErrorContext(ctx, "failed to create metrics FIFO",
			slog.String("vm_id", vmID),
			slog.String("path", hostMetricsPath),
			slog.String("error", err.Error()),
		)
	} else {
		c.logger.InfoContext(ctx, "created metrics FIFO for billaged",
			slog.String("vm_id", vmID),
			slog.String("host_path", hostMetricsPath),
			slog.String("chroot_path", metricsPath),
		)
	}

	return metricsPath
}

// buildSDKConfig builds the base SDK configuration
func (c *Client) buildSDKConfig(
	socketPath string,
	consoleLogPath string,
	consoleFifoPath string,
	metricsPath string,
	kernelPath string,
	kernelArgs string,
	config *metaldv1.VmConfig,
	jailerRoot string,
) sdk.Config {
	// Create the console log file to capture guest output
	consoleLogFile, err := os.OpenFile(consoleLogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)

	var cfg sdk.Config
	if err != nil {
		// Fall back to LogPath only if console log file creation fails
		c.logger.Warn("failed to create console log file, falling back to LogPath only",
			slog.String("error", err.Error()),
			slog.String("console_log_path", consoleLogPath),
		)
		cfg = sdk.Config{
			SocketPath:      socketPath,
			LogPath:         consoleLogPath, // Captures Firecracker logs only
			LogLevel:        "Debug",
			MetricsPath:     metricsPath,
			KernelImagePath: kernelPath,
			KernelArgs:      kernelArgs,
			MachineCfg: models.MachineConfiguration{
				VcpuCount:  sdk.Int64(int64(config.GetVcpuCount())),
				MemSizeMib: sdk.Int64(536870912),
				Smt:        sdk.Bool(false),
			},
		}
	} else {
		// Successful case - capture guest console output via FIFO
		cfg = sdk.Config{
			SocketPath:      socketPath,
			LogPath:         filepath.Join(jailerRoot, "firecracker.log"), // Firecracker's own logs
			LogFifo:         consoleFifoPath,                              // FIFO for guest console output
			FifoLogWriter:   consoleLogFile,                               // Writer to capture guest console to file
			LogLevel:        "Debug",
			MetricsPath:     metricsPath,
			KernelImagePath: kernelPath,
			KernelArgs:      kernelArgs,
			MachineCfg: models.MachineConfiguration{
				VcpuCount:  sdk.Int64(int64(config.GetVcpuCount())),
				MemSizeMib: sdk.Int64(536870912),
				Smt:        sdk.Bool(false),
			},
		}
	}

	return cfg
}

// addNetworkInterfaceToConfig adds network interface to the Firecracker configuration
func (c *Client) addNetworkInterfaceToConfig(cfg *sdk.Config, networkInfo *network.VMNetwork) {
	iface := sdk.NetworkInterface{
		StaticConfiguration: &sdk.StaticNetworkConfiguration{
			HostDevName: networkInfo.TapDevice,
			MacAddress:  networkInfo.MacAddress,
		},
	}
	cfg.NetworkInterfaces = []sdk.NetworkInterface{iface}
}
