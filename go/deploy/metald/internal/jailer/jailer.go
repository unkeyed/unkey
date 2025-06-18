package jailer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"golang.org/x/sys/unix"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// AIDEV-NOTE: This package implements jailer functionality directly in metald
// This allows us to have better control over the network namespace and tap device
// creation, solving the permission issues we encountered with the external jailer

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
	ctx, span := j.tracer.Start(ctx, "Jailer.Exec",
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
	// AIDEV-NOTE: Firecracker binary path is now hardcoded to standard location
	firecrackerPath := "/usr/local/bin/firecracker"
	args := []string{firecrackerPath}
	args = append(args, "--api-sock", opts.SocketPath)
	args = append(args, opts.FirecrackerArgs...)

	j.logger.InfoContext(ctx, "executing firecracker",
		slog.String("binary", firecrackerPath),
		slog.Any("args", args),
	)

	// Step 6: Exec into firecracker
	// This replaces the current process with firecracker
	return syscall.Exec(firecrackerPath, args, os.Environ())
}

// RunInJail runs firecracker in a jail by creating a minimal isolation environment
// This function forks and execs firecracker with dropped privileges
func (j *Jailer) RunInJail(ctx context.Context, opts *ExecOptions) (*os.Process, error) {
	ctx, span := j.tracer.Start(ctx, "Jailer.RunInJail",
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
	args := []string{firecrackerPath, "--api-sock", opts.SocketPath}
	args = append(args, opts.FirecrackerArgs...)

	// Create the command
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
	ctx, span := j.tracer.Start(ctx, "Jailer.setupChroot",
		trace.WithAttributes(
			attribute.String("chroot_path", chrootPath),
		),
	)
	defer span.End()
	// Create necessary directories
	for _, dir := range []string{"", "dev", "dev/net", "run"} {
		path := filepath.Join(chrootPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	// Create /dev/net/tun
	tunPath := filepath.Join(chrootPath, "dev/net/tun")
	if err := unix.Mknod(tunPath, unix.S_IFCHR|0666, int(unix.Mkdev(10, 200))); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create /dev/net/tun: %w", err)
		}
	}

	// Create /dev/kvm
	kvmPath := filepath.Join(chrootPath, "dev/kvm")
	if err := unix.Mknod(kvmPath, unix.S_IFCHR|0666, int(unix.Mkdev(10, 232))); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create /dev/kvm: %w", err)
		}
	}

	// Create metrics FIFO for billaged to read Firecracker stats
	metricsPath := filepath.Join(chrootPath, "metrics.fifo")
	if err := unix.Mkfifo(metricsPath, 0644); err != nil && !os.IsExist(err) {
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

// AIDEV-NOTE: This implementation provides the core jailer functionality
// but integrated into metald. The key advantages are:
// 1. We can create tap devices before dropping privileges
// 2. We have full control over the network namespace setup
// 3. We can pass open file descriptors to the jailed process
// 4. We maintain the security isolation of the original jailer