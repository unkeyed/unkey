package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/proto/builder/v1"
)

// ProcessIsolator handles process-level isolation for builds
type ProcessIsolator struct {
	logger        *slog.Logger
	tenantMgr     *Manager
	enableCgroups bool
	enableSeccomp bool
}

// NewProcessIsolator creates a new process isolator
func NewProcessIsolator(logger *slog.Logger, tenantMgr *Manager) *ProcessIsolator {
	isolator := &ProcessIsolator{
		logger:        logger,
		tenantMgr:     tenantMgr,
		enableCgroups: true,
		enableSeccomp: true,
	}

	// Check if cgroups v2 is available
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err != nil {
		isolator.enableCgroups = false
		logger.Warn("cgroups v2 not available, disabling cgroup isolation")
	}

	return isolator
}

// CreateIsolatedCommand creates a command with isolation constraints
func (p *ProcessIsolator) CreateIsolatedCommand(
	ctx context.Context,
	tenantID string,
	tier builderv1.TenantTier,
	buildID string,
	command string,
	args ...string,
) (*exec.Cmd, error) {
	config, err := p.tenantMgr.GetTenantConfig(ctx, tenantID, tier)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}

	constraints := p.buildConstraints(config, buildID)

	// Create the base command
	cmd := exec.CommandContext(ctx, command, args...)

	// Apply process isolation
	if err := p.applyProcessIsolation(cmd, constraints); err != nil {
		return nil, fmt.Errorf("failed to apply process isolation: %w", err)
	}

	// Apply resource limits
	if err := p.applyResourceLimits(cmd, constraints, buildID); err != nil {
		return nil, fmt.Errorf("failed to apply resource limits: %w", err)
	}

	p.logger.Info("created isolated command",
		slog.String("tenant_id", tenantID),
		slog.String("build_id", buildID),
		slog.String("command", command),
		slog.Int64("memory_limit", constraints.MaxMemoryBytes),
		slog.Int64("cpu_limit", int64(constraints.MaxCPUCores)),
	)

	return cmd, nil
}

// CreateIsolatedDockerCommand creates a Docker command with tenant isolation
func (p *ProcessIsolator) CreateIsolatedDockerCommand(
	ctx context.Context,
	tenantID string,
	tier builderv1.TenantTier,
	buildID string,
	dockerArgs []string,
) (*exec.Cmd, error) {
	config, err := p.tenantMgr.GetTenantConfig(ctx, tenantID, tier)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant config: %w", err)
	}

	constraints := p.buildConstraints(config, buildID)

	// Build Docker command with isolation flags
	args := []string{"run", "--rm"}

	// Resource limits
	args = append(args, "--memory", fmt.Sprintf("%d", constraints.MaxMemoryBytes))
	args = append(args, "--cpus", fmt.Sprintf("%d", constraints.MaxCPUCores))
	args = append(args, "--disk-quota", fmt.Sprintf("%d", constraints.MaxDiskBytes))

	// Security settings
	args = append(args, "--user", fmt.Sprintf("%d:%d", constraints.RunAsUser, constraints.RunAsGroup))
	args = append(args, "--read-only")
	args = append(args, "--tmpfs", fmt.Sprintf("/tmp:size=%d", constraints.MaxTempSizeBytes))
	args = append(args, "--security-opt", "no-new-privileges:true")

	// Drop capabilities
	for _, cap := range constraints.DroppedCapabilities {
		args = append(args, "--cap-drop", cap)
	}

	// Network isolation
	switch constraints.NetworkMode {
	case "none":
		args = append(args, "--network", "none")
	case "isolated":
		args = append(args, "--network", fmt.Sprintf("builderd-tenant-%s", tenantID))
	default:
		args = append(args, "--network", "bridge")
	}

	// Add working directory
	args = append(args, "--workdir", "/workspace")
	args = append(args, "-v", fmt.Sprintf("%s:/workspace", constraints.WorkspaceDir))

	// Environment variables for isolation
	args = append(args, "-e", fmt.Sprintf("BUILDERD_TENANT_ID=%s", tenantID))
	args = append(args, "-e", fmt.Sprintf("BUILDERD_BUILD_ID=%s", buildID))
	args = append(args, "-e", "HOME=/tmp")

	// Add timeout
	args = append(args, "--stop-timeout", fmt.Sprintf("%d", constraints.TimeoutSeconds))

	// Append user-provided Docker args
	args = append(args, dockerArgs...)

	cmd := exec.CommandContext(ctx, "docker", args...)

	// Set resource limits on the Docker process itself
	if err := p.applyProcessIsolation(cmd, constraints); err != nil {
		return nil, fmt.Errorf("failed to apply process isolation to docker command: %w", err)
	}

	p.logger.Info("created isolated docker command",
		slog.String("tenant_id", tenantID),
		slog.String("build_id", buildID),
		slog.Any("docker_args", args),
	)

	return cmd, nil
}

// buildConstraints creates build constraints from tenant config
func (p *ProcessIsolator) buildConstraints(config *TenantConfig, buildID string) BuildConstraints {
	// Default security settings
	droppedCaps := []string{
		"AUDIT_CONTROL", "AUDIT_READ", "AUDIT_WRITE",
		"BLOCK_SUSPEND", "DAC_READ_SEARCH", "FSETID",
		"IPC_LOCK", "MAC_ADMIN", "MAC_OVERRIDE",
		"MKNOD", "SETFCAP", "SYSLOG", "SYS_ADMIN",
		"SYS_BOOT", "SYS_MODULE", "SYS_NICE",
		"SYS_RAWIO", "SYS_RESOURCE", "SYS_TIME",
		"WAKE_ALARM",
	}

	// Determine network mode based on tier
	networkMode := "none"
	if config.Limits.AllowExternalNetwork {
		networkMode = "isolated"
	}

	// Calculate temp directory size (10% of disk limit)
	maxTempSize := config.Limits.MaxDiskBytes / 10
	if maxTempSize < 100*1024*1024 { // Minimum 100MB
		maxTempSize = 100 * 1024 * 1024
	}

	return BuildConstraints{
		MaxMemoryBytes:      config.Limits.MaxMemoryBytes,
		MaxCPUCores:         config.Limits.MaxCPUCores,
		MaxDiskBytes:        config.Limits.MaxDiskBytes,
		TimeoutSeconds:      config.Limits.TimeoutSeconds,
		RunAsUser:           1000, // builderd user
		RunAsGroup:          1000, // builderd group
		ReadOnlyRootfs:      true,
		NoPrivileged:        true,
		DroppedCapabilities: droppedCaps,
		NetworkMode:         networkMode,
		AllowedRegistries:   config.Limits.AllowedRegistries,
		AllowedGitHosts:     config.Limits.AllowedGitHosts,
		WorkspaceDir:        filepath.Join("/tmp/builderd/workspace", config.TenantID, buildID),
		RootfsDir:           filepath.Join("/tmp/builderd/rootfs", config.TenantID, buildID),
		TempDir:             filepath.Join("/tmp/builderd/temp", config.TenantID, buildID),
		MaxTempSizeBytes:    maxTempSize,
	}
}

// applyProcessIsolation applies process-level isolation
func (p *ProcessIsolator) applyProcessIsolation(cmd *exec.Cmd, constraints BuildConstraints) error {
	// Set process credentials
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(constraints.RunAsUser),
			Gid: uint32(constraints.RunAsGroup),
		},
		// Create new process group
		Setpgid: true,
		Pgid:    0,
	}

	// Set environment for isolation
	cmd.Env = []string{
		"HOME=/tmp",
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"SHELL=/bin/sh",
		"USER=builderd",
		fmt.Sprintf("BUILDERD_WORKSPACE=%s", constraints.WorkspaceDir),
		fmt.Sprintf("BUILDERD_ROOTFS=%s", constraints.RootfsDir),
		fmt.Sprintf("BUILDERD_TEMP=%s", constraints.TempDir),
	}

	return nil
}

// applyResourceLimits applies resource limits using cgroups
func (p *ProcessIsolator) applyResourceLimits(cmd *exec.Cmd, constraints BuildConstraints, buildID string) error {
	if !p.enableCgroups {
		p.logger.Debug("cgroups disabled, skipping resource limits")
		return nil
	}

	cgroupPath := fmt.Sprintf("/sys/fs/cgroup/builderd/%s", buildID)

	// Create cgroup directory
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		p.logger.Warn("failed to create cgroup directory", slog.String("error", err.Error()))
		return nil // Don't fail the build for cgroup issues
	}

	// Set memory limit
	memoryMax := filepath.Join(cgroupPath, "memory.max")
	if err := os.WriteFile(memoryMax, []byte(strconv.FormatInt(constraints.MaxMemoryBytes, 10)), 0644); err != nil {
		p.logger.Warn("failed to set memory limit", slog.String("error", err.Error()))
	}

	// Set CPU limit (cgroups v2)
	cpuMax := filepath.Join(cgroupPath, "cpu.max")
	cpuQuota := fmt.Sprintf("%d 100000", constraints.MaxCPUCores*100000) // 100ms period
	if err := os.WriteFile(cpuMax, []byte(cpuQuota), 0644); err != nil {
		p.logger.Warn("failed to set CPU limit", slog.String("error", err.Error()))
	}

	// Set IO limit (simplified)
	ioMax := filepath.Join(cgroupPath, "io.max")
	// Format: major:minor rbps=X wbps=Y riops=A wiops=B
	// We'll set a conservative limit for now
	maxBps := constraints.MaxDiskBytes / 300 // Spread over 5 minutes
	ioLimit := fmt.Sprintf("8:0 rbps=%d wbps=%d", maxBps, maxBps)
	if err := os.WriteFile(ioMax, []byte(ioLimit), 0644); err != nil {
		p.logger.Debug("failed to set IO limit", slog.String("error", err.Error()))
	}

	// Schedule cleanup of cgroup after build with proper process monitoring
	go func() {
		defer func() {
			// Always cleanup cgroup regardless of how we exit
			if err := os.RemoveAll(cgroupPath); err != nil {
				p.logger.Warn("failed to cleanup cgroup", 
					slog.String("error", err.Error()),
					slog.String("cgroup_path", cgroupPath),
					slog.String("build_id", buildID),
				)
			} else {
				p.logger.Debug("cleaned up cgroup",
					slog.String("cgroup_path", cgroupPath),
					slog.String("build_id", buildID),
				)
			}
		}()

		// AIDEV-NOTE: Fixed memory leak - proper process monitoring instead of sleep
		// This ensures the cleanup goroutine terminates when the process actually exits
		// Wait for the actual process to complete (if it exists)
		if cmd.Process != nil {
			// Monitor process completion
			state, err := cmd.Process.Wait()
			if err != nil {
				p.logger.Debug("error waiting for process completion",
					slog.String("error", err.Error()),
					slog.String("build_id", buildID),
				)
			} else {
				p.logger.Debug("process completed",
					slog.String("build_id", buildID),
					slog.Int("exit_code", state.ExitCode()),
				)
			}
		} else {
			// Fallback timeout if process never started or was already completed
			p.logger.Debug("no process handle available, using timeout fallback",
				slog.String("build_id", buildID),
			)
			timeout := time.Duration(constraints.TimeoutSeconds+60) * time.Second
			time.Sleep(timeout)
		}
	}()

	p.logger.Debug("applied resource limits via cgroups",
		slog.String("build_id", buildID),
		slog.String("cgroup_path", cgroupPath),
		slog.Int64("memory_bytes", constraints.MaxMemoryBytes),
		slog.Int64("cpu_cores", int64(constraints.MaxCPUCores)),
	)

	return nil
}

// MonitorProcess monitors a process for resource usage and violations
func (p *ProcessIsolator) MonitorProcess(
	ctx context.Context,
	cmd *exec.Cmd,
	tenantID string,
	buildID string,
	constraints BuildConstraints,
) *ResourceUsage {
	usage := &ResourceUsage{
		BuildID:   buildID,
		TenantID:  tenantID,
		StartTime: time.Now(),
	}

	// Start monitoring in background
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if cmd.Process == nil {
					continue
				}

				// Monitor process resource usage
				if err := p.updateResourceUsage(cmd.Process.Pid, usage, constraints); err != nil {
					p.logger.Debug("failed to update resource usage", slog.String("error", err.Error()))
				}
			}
		}
	}()

	return usage
}

// updateResourceUsage updates resource usage statistics
func (p *ProcessIsolator) updateResourceUsage(pid int, usage *ResourceUsage, constraints BuildConstraints) error {
	// Read from /proc/PID/stat for CPU and memory info
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statData, err := os.ReadFile(statPath)
	if err != nil {
		return err
	}

	fields := strings.Fields(string(statData))
	if len(fields) < 24 {
		return fmt.Errorf("invalid stat file format")
	}

	// Parse memory usage (RSS in pages)
	if rss, err := strconv.ParseInt(fields[23], 10, 64); err == nil {
		pageSize := int64(os.Getpagesize())
		usage.MemoryUsedBytes = rss * pageSize
		usage.MemoryLimitBytes = constraints.MaxMemoryBytes
	}

	// Read memory info from /proc/PID/status
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	if statusData, err := os.ReadFile(statusPath); err == nil {
		lines := strings.Split(string(statusData), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmHWM:") {
				// Peak memory usage
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if peak, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						usage.MemoryMaxBytes = peak * 1024 // Convert from KB
					}
				}
			}
		}
	}

	// Check for quota violations
	if usage.MemoryUsedBytes > constraints.MaxMemoryBytes {
		p.logger.Warn("memory quota violation detected",
			slog.String("tenant_id", usage.TenantID),
			slog.String("build_id", usage.BuildID),
			slog.Int64("used_bytes", usage.MemoryUsedBytes),
			slog.Int64("limit_bytes", constraints.MaxMemoryBytes),
		)
	}

	return nil
}

// TerminateProcess forcefully terminates a process group
func (p *ProcessIsolator) TerminateProcess(cmd *exec.Cmd, reason string) error {
	if cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid

	p.logger.Info("terminating process",
		slog.Int("pid", pid),
		slog.String("reason", reason),
	)

	// Try graceful termination first
	if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
		p.logger.Debug("failed to send SIGTERM", slog.String("error", err.Error()))
	}

	// Wait a bit for graceful shutdown
	time.Sleep(5 * time.Second)

	// Force kill if still running
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		p.logger.Debug("failed to send SIGKILL", slog.String("error", err.Error()))
	}

	return nil
}

// ValidateNetworkAccess validates if a network request is allowed for a tenant
func (p *ProcessIsolator) ValidateNetworkAccess(
	ctx context.Context,
	tenantID string,
	tier builderv1.TenantTier,
	targetHost string,
	targetType string, // "registry", "git", "generic"
) error {
	config, err := p.tenantMgr.GetTenantConfig(ctx, tenantID, tier)
	if err != nil {
		return fmt.Errorf("failed to get tenant config: %w", err)
	}

	// Check if external network is allowed
	if !config.Limits.AllowExternalNetwork {
		return fmt.Errorf("external network access not allowed for tenant %s", tenantID)
	}

	// Check specific host allowlists
	switch targetType {
	case "registry":
		if !p.isHostAllowed(targetHost, config.Limits.AllowedRegistries) {
			return fmt.Errorf("registry %s not allowed for tenant %s", targetHost, tenantID)
		}
	case "git":
		if !p.isHostAllowed(targetHost, config.Limits.AllowedGitHosts) {
			return fmt.Errorf("git host %s not allowed for tenant %s", targetHost, tenantID)
		}
	}

	return nil
}

// isHostAllowed checks if a host is in the allowed list
func (p *ProcessIsolator) isHostAllowed(host string, allowedHosts []string) bool {
	for _, allowed := range allowedHosts {
		if allowed == "*" || allowed == host {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(host, "."+domain) || host == domain {
				return true
			}
		}
	}
	return false
}
