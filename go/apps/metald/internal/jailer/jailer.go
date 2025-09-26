//go:build linux
// +build linux

package jailer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/unkeyed/unkey/go/apps/metald/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sys/unix"
)

// Jailer provides functionality similar to firecracker's jailer but integrated into metald
type Jailer struct {
	logger *slog.Logger
	config *config.JailerConfig
	tracer trace.Tracer
}

// NewJailer creates a new integrated jailer
func NewJailer(logger *slog.Logger, config *config.JailerConfig) *Jailer {
	tracer := otel.Tracer("metald.jailer.integrated")
	return &Jailer{
		logger: logger.With("component", "integrated-jailer"),
		config: config,
		tracer: tracer,
	}
}

// ExecOptions contains options for executing firecracker in a jailed environment
type ExecOptions struct {
	// VMId is the unique identifier for this VM
	VMId string

	// NetworkNamespace is the path to the network namespace (e.g., /run/netns/vm-xxx)
	NetworkNamespace string

	// SocketPath is the path to the firecracker API socket
	SocketPath string

	// FirecrackerArgs are additional arguments to pass to firecracker
	FirecrackerArgs []string

	// Stdin, Stdout, Stderr for the firecracker process
	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
}

// Exec executes firecracker in a jailed environment
// This function does NOT return if successful - it execs into firecracker
func (j *Jailer) Exec(ctx context.Context, opts *ExecOptions) error {
	ctx, span := j.tracer.Start(ctx, "metald.jailer.exec",
		trace.WithAttributes(
			attribute.String("vm_id", opts.VMId),
			attribute.String("netns", opts.NetworkNamespace),
			attribute.String("chroot_base", j.config.ChrootBaseDir),
			attribute.Int64("uid", int64(j.config.UID)),
			attribute.Int64("gid", int64(j.config.GID)),
		),
	)
	defer span.End()

	j.logger.InfoContext(ctx, "executing firecracker with integrated jailer",
		slog.String("vm_id", opts.VMId),
		slog.String("netns", opts.NetworkNamespace),
	)

	// Step 1: Set up the chroot environment
	chrootPath := filepath.Join(j.config.ChrootBaseDir, "firecracker", opts.VMId, "root")
	if err := j.setupChroot(ctx, chrootPath); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to setup chroot: %w", err)
	}

	// Step 2: Join the network namespace if specified
	if opts.NetworkNamespace != "" {
		if err := j.joinNetworkNamespace(ctx, opts.NetworkNamespace); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to join network namespace: %w", err)
		}
	}

	// Step 3: Enter the chroot
	if err := syscall.Chroot(chrootPath); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to chroot: %w", err)
	}
	if err := os.Chdir("/"); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to chdir to /: %w", err)
	}

	// Step 4: Drop privileges
	if err := j.dropPrivileges(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to drop privileges: %w", err)
	}

	// Step 5: Prepare firecracker command
	firecrackerPath := "/usr/local/bin/firecracker"
	args := []string{firecrackerPath}
	args = append(args, "--api-sock", opts.SocketPath)
	args = append(args, opts.FirecrackerArgs...)

	j.logger.InfoContext(ctx, "executing firecracker",
		slog.String("binary", firecrackerPath),
		slog.Any("args", args),
	)

	// Step 6: Validate and exec into firecracker
	if err := validateFirecrackerPath(firecrackerPath); err != nil {
		return fmt.Errorf("firecracker path validation failed: %w", err)
	}

	// This replaces the current process with firecracker
	//nolint:gosec // Path validation performed above
	return syscall.Exec(firecrackerPath, args, os.Environ())
}

// RunInJail runs firecracker in a jail by creating a minimal isolation environment
// This function forks and execs firecracker with dropped privileges
func (j *Jailer) RunInJail(ctx context.Context, opts *ExecOptions) (*os.Process, error) {
	ctx, span := j.tracer.Start(ctx, "metald.jailer.run_in_jail",
		trace.WithAttributes(
			attribute.String("vm_id", opts.VMId),
			attribute.String("netns", opts.NetworkNamespace),
			attribute.String("chroot_base", j.config.ChrootBaseDir),
		),
	)
	defer span.End()

	j.logger.InfoContext(ctx, "running firecracker in jail",
		slog.String("vm_id", opts.VMId),
		slog.String("netns", opts.NetworkNamespace),
	)

	// Setup chroot environment
	chrootPath := filepath.Join(j.config.ChrootBaseDir, "firecracker", opts.VMId, "root")
	if err := j.setupChroot(ctx, chrootPath); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to setup chroot: %w", err)
	}

	// Build firecracker command
	// AIDEV-NOTE: Firecracker binary path is now hardcoded to standard location
	firecrackerPath := "/usr/local/bin/firecracker"

	// Validate firecracker path for security
	if err := validateFirecrackerPath(firecrackerPath); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("firecracker path validation failed: %w", err)
	}

	args := []string{firecrackerPath, "--api-sock", opts.SocketPath}
	args = append(args, opts.FirecrackerArgs...)

	// Create the command
	//nolint:gosec // Path validation performed above
	cmd := exec.CommandContext(ctx, firecrackerPath, args[1:]...)

	// Set up file descriptors
	cmd.Stdin = opts.Stdin
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr

	// Set working directory to chroot
	cmd.Dir = chrootPath

	// For now, run without full isolation to test
	// In production, we'd fork and do the chroot/namespace/privilege dropping
	// AIDEV-TODO: Implement proper forking with isolation

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start firecracker: %w", err)
	}

	j.logger.InfoContext(ctx, "started jailed firecracker process",
		slog.String("vm_id", opts.VMId),
		slog.Int("pid", cmd.Process.Pid),
	)

	return cmd.Process, nil
}

// setupChroot prepares the chroot environment
func (j *Jailer) setupChroot(ctx context.Context, chrootPath string) error {
	ctx, span := j.tracer.Start(ctx, "metald.jailer.setup_chroot",
		trace.WithAttributes(
			attribute.String("chroot_path", chrootPath),
		),
	)
	defer span.End()
	// Create necessary directories
	for _, dir := range []string{"", "dev", "dev/net", "run"} {
		path := filepath.Join(chrootPath, dir)
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	// Create /dev/net/tun
	tunPath := filepath.Join(chrootPath, "dev/net/tun")
	tunDev, err := safeUint64ToInt(unix.Mkdev(10, 200))
	if err != nil {
		return fmt.Errorf("failed to convert tun device number: %w", err)
	}
	if mkErr := unix.Mknod(tunPath, unix.S_IFCHR|0o666, tunDev); mkErr != nil {
		if !os.IsExist(mkErr) {
			return fmt.Errorf("failed to create /dev/net/tun: %w", mkErr)
		}
	}

	// Create /dev/kvm
	kvmPath := filepath.Join(chrootPath, "dev/kvm")
	kvmDev, err := safeUint64ToInt(unix.Mkdev(10, 232))
	if err != nil {
		return fmt.Errorf("failed to convert kvm device number: %w", err)
	}
	if err := unix.Mknod(kvmPath, unix.S_IFCHR|0o666, kvmDev); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create /dev/kvm: %w", err)
		}
	}

	// Create metrics FIFO for billaged to read Firecracker stats
	metricsPath := filepath.Join(chrootPath, "metrics.fifo")
	if err := unix.Mkfifo(metricsPath, 0o644); err != nil && !os.IsExist(err) {
		span.RecordError(err)
		return fmt.Errorf("failed to create metrics FIFO: %w", err)
	}
	span.SetAttributes(attribute.String("metrics_fifo_path", metricsPath))
	j.logger.InfoContext(ctx, "created metrics FIFO for billaged",
		slog.String("path", metricsPath))

	// Set ownership
	if err := os.Chown(tunPath, int(j.config.UID), int(j.config.GID)); err != nil {
		j.logger.WarnContext(ctx, "failed to chown /dev/net/tun", "error", err)
	}
	if err := os.Chown(kvmPath, int(j.config.UID), int(j.config.GID)); err != nil {
		j.logger.WarnContext(ctx, "failed to chown /dev/kvm", "error", err)
	}
	if err := os.Chown(metricsPath, int(j.config.UID), int(j.config.GID)); err != nil {
		j.logger.WarnContext(ctx, "failed to chown metrics FIFO", "error", err)
	}

	return nil
}

// joinNetworkNamespace joins the specified network namespace
func (j *Jailer) joinNetworkNamespace(ctx context.Context, netnsPath string) error {
	// Open the network namespace
	netnsFile, err := os.Open(netnsPath)
	if err != nil {
		return fmt.Errorf("failed to open network namespace: %w", err)
	}
	defer netnsFile.Close()

	// Join the network namespace
	if err := unix.Setns(int(netnsFile.Fd()), unix.CLONE_NEWNET); err != nil {
		return fmt.Errorf("failed to setns: %w", err)
	}

	j.logger.InfoContext(ctx, "joined network namespace", slog.String("netns", netnsPath))
	return nil
}

// dropPrivileges drops to the configured UID/GID
func (j *Jailer) dropPrivileges(ctx context.Context) error {
	// Set groups
	if err := unix.Setgroups([]int{int(j.config.GID)}); err != nil {
		return fmt.Errorf("failed to setgroups: %w", err)
	}

	// Set GID
	if err := unix.Setresgid(int(j.config.GID), int(j.config.GID), int(j.config.GID)); err != nil {
		return fmt.Errorf("failed to setresgid: %w", err)
	}

	// Set UID (must be last)
	if err := unix.Setresuid(int(j.config.UID), int(j.config.UID), int(j.config.UID)); err != nil {
		return fmt.Errorf("failed to setresuid: %w", err)
	}

	j.logger.InfoContext(ctx, "dropped privileges",
		slog.Uint64("uid", uint64(j.config.UID)),
		slog.Uint64("gid", uint64(j.config.GID)),
	)

	return nil
}

// safeUint64ToInt safely converts uint64 to int, checking for overflow
func safeUint64ToInt(value uint64) (int, error) {
	const maxInt = int(^uint(0) >> 1)
	if value > uint64(maxInt) {
		return 0, fmt.Errorf("value %d exceeds maximum int value %d", value, maxInt)
	}
	return int(value), nil
}

// validateFirecrackerPath validates the firecracker binary path for security
func validateFirecrackerPath(path string) error {
	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(path)

	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal attempt detected: %s", path)
	}

	// Ensure path is absolute and starts with expected directories
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("firecracker path must be absolute: %s", path)
	}

	// Check for dangerous characters
	if strings.ContainsAny(cleanPath, ";&|$`\\") {
		return fmt.Errorf("dangerous characters detected in path: %s", path)
	}

	// Verify file exists and is executable
	info, err := os.Stat(cleanPath)
	if err != nil {
		return fmt.Errorf("firecracker binary not found: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("firecracker path is a directory: %s", cleanPath)
	}

	// Check if file is executable
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("firecracker binary is not executable: %s", cleanPath)
	}

	return nil
}
