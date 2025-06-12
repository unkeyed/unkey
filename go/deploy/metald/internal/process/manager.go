package process

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// FirecrackerProcess represents a managed Firecracker instance
type FirecrackerProcess struct {
	ID                string
	SocketPath        string
	PIDFile           string
	LogFile           string
	Process           *os.Process
	Started           time.Time
	Status            ProcessStatus
	VMID              string // VM currently assigned to this process
	MetricsConfigured bool   // Whether metrics have been configured for this process

	// Jailer specific fields
	UseJailer  bool   // Whether this process uses jailer
	JailerID   string // Unique jailer ID for chroot isolation
	ChrootPath string // Path to jailer chroot directory
}

type ProcessStatus string

const (
	StatusStarting ProcessStatus = "starting"
	StatusReady    ProcessStatus = "ready"
	StatusBusy     ProcessStatus = "busy"
	StatusStopping ProcessStatus = "stopping"
	StatusStopped  ProcessStatus = "stopped"
	StatusError    ProcessStatus = "error"
)

// Manager manages Firecracker processes
type Manager struct {
	logger    *slog.Logger
	processes map[string]*FirecrackerProcess
	mutex     sync.RWMutex

	// Application context for long-lived processes (separate from request contexts)
	// AIDEV-NOTE: This context enables tracing while keeping processes alive beyond requests
	appCtx context.Context

	// Configuration
	socketDir    string
	logDir       string
	maxProcesses int
	idleTimeout  time.Duration
	startTimeout time.Duration

	// Jailer configuration
	jailerConfig *config.JailerConfig
}

// NewManager creates a new Firecracker process manager
func NewManager(logger *slog.Logger, appCtx context.Context, pmConfig *config.ProcessManagerConfig) *Manager {
	return &Manager{
		logger:       logger.With("component", "process_manager"),
		processes:    make(map[string]*FirecrackerProcess),
		appCtx:       appCtx, // Long-lived context for process lifecycle
		socketDir:    pmConfig.SocketDir,
		logDir:       pmConfig.LogDir,
		maxProcesses: pmConfig.MaxProcesses,
		idleTimeout:  5 * time.Minute,
		startTimeout: 30 * time.Second,
		jailerConfig: nil, // Will be set via SetJailerConfig
	}
}

// NewManagerWithConfig creates a new Firecracker process manager with jailer config
func NewManagerWithConfig(logger *slog.Logger, appCtx context.Context, pmConfig *config.ProcessManagerConfig, jailerConfig *config.JailerConfig) *Manager {
	manager := NewManager(logger, appCtx, pmConfig)
	manager.jailerConfig = jailerConfig
	return manager
}

// SetJailerConfig sets the jailer configuration
func (m *Manager) SetJailerConfig(jailerConfig *config.JailerConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.jailerConfig = jailerConfig
}

// Initialize sets up the process manager
func (m *Manager) Initialize() error {
	m.logger.Info("initializing process manager")

	// Create directories
	if err := os.MkdirAll(m.socketDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if err := os.MkdirAll(m.logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Clean up any leftover sockets from previous runs
	m.cleanupLeftoverSockets()

	m.logger.Info("process manager initialized",
		slog.String("socket_dir", m.socketDir),
		slog.String("log_dir", m.logDir),
		slog.Int("max_processes", m.maxProcesses),
	)

	return nil
}

// GetOrCreateProcess creates a dedicated new process for the VM (no reuse)
func (m *Manager) GetOrCreateProcess(ctx context.Context, vmID string) (*FirecrackerProcess, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.LogAttrs(ctx, slog.LevelInfo, "creating dedicated process for vm",
		slog.String("vm_id", vmID),
	)

	// Check if VM already has a process assigned
	for _, proc := range m.processes {
		if proc.VMID == vmID {
			m.logger.LogAttrs(ctx, slog.LevelInfo, "found existing process for vm",
				slog.String("vm_id", vmID),
				slog.String("process_id", proc.ID),
				slog.String("status", string(proc.Status)),
			)
			return proc, nil
		}
	}

	// Always create a new dedicated process for each VM
	if len(m.processes) >= m.maxProcesses {
		return nil, fmt.Errorf("maximum number of processes (%d) reached", m.maxProcesses)
	}

	process, err := m.createNewProcess(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new process: %w", err)
	}

	// Assign VM to process
	process.VMID = vmID
	process.Status = StatusBusy

	m.logger.LogAttrs(ctx, slog.LevelInfo, "created dedicated process for vm",
		slog.String("vm_id", vmID),
		slog.String("process_id", process.ID),
		slog.String("socket_path", process.SocketPath),
	)

	return process, nil
}

// ReleaseProcess terminates the dedicated process when VM is deleted
func (m *Manager) ReleaseProcess(ctx context.Context, vmID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for _, proc := range m.processes {
		if proc.VMID == vmID {
			m.logger.LogAttrs(ctx, slog.LevelInfo, "terminating dedicated process",
				slog.String("vm_id", vmID),
				slog.String("process_id", proc.ID),
			)

			// Always terminate the process (no reuse in 1:1 model)
			return m.stopProcess(ctx, proc)
		}
	}

	return fmt.Errorf("no process found for vm %s", vmID)
}

// StopProcess stops a specific Firecracker process
func (m *Manager) StopProcess(ctx context.Context, processID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	proc, exists := m.processes[processID]
	if !exists {
		return fmt.Errorf("process %s not found", processID)
	}

	return m.stopProcess(ctx, proc)
}

// Shutdown stops all processes and cleans up
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.logger.Info("shutting down process manager")

	for _, proc := range m.processes {
		if err := m.stopProcess(ctx, proc); err != nil {
			m.logger.LogAttrs(ctx, slog.LevelError, "failed to stop process",
				slog.String("process_id", proc.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	m.cleanupLeftoverSockets()

	m.logger.Info("process manager shutdown complete")
	return nil
}

// createNewProcess spawns a new Firecracker process
func (m *Manager) createNewProcess(ctx context.Context) (*FirecrackerProcess, error) {
	processID := fmt.Sprintf("fc-%d", time.Now().UnixNano())

	// Check if jailer is enabled
	useJailer := m.jailerConfig != nil && m.jailerConfig.Enabled
	var jailerID string
	var chrootPath string
	var socketPath string

	if useJailer {
		// Generate unique jailer ID (max 64 chars as per jailer requirements)
		jailerID = fmt.Sprintf("vm-%d", time.Now().UnixNano())
		chrootPath = filepath.Join(m.jailerConfig.ChrootBaseDir, jailerID)
		// Jailer creates socket inside chroot at /tmp/firecracker.socket
		socketPath = filepath.Join(chrootPath, "root/tmp/firecracker.socket")
	} else {
		socketPath = filepath.Join(m.socketDir, processID+".sock")
	}

	process := &FirecrackerProcess{
		ID:         processID,
		SocketPath: socketPath,
		PIDFile:    filepath.Join(m.logDir, processID+".pid"),
		LogFile:    filepath.Join(m.logDir, processID+".log"),
		Started:    time.Now(),
		Status:     StatusStarting,
		UseJailer:  useJailer,
		JailerID:   jailerID,
		ChrootPath: chrootPath,
	}

	m.logger.LogAttrs(ctx, slog.LevelInfo, "creating new firecracker process",
		slog.String("process_id", processID),
		slog.String("socket_path", process.SocketPath),
		slog.Bool("use_jailer", useJailer),
		slog.String("jailer_id", jailerID),
	)

	// Create log file
	logFile, err := os.Create(process.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Start process with application context for tracing, not request context
	// AIDEV-NOTE: Use appCtx (application-scoped) instead of ctx (request-scoped) for process lifecycle
	// This enables distributed tracing while keeping processes alive beyond individual requests
	processCtx := m.createProcessContext(ctx, processID)

	var cmd *exec.Cmd
	if useJailer {
		cmd, err = m.createJailerCommand(processCtx, process)
		if err != nil {
			return nil, fmt.Errorf("failed to create jailer command: %w", err)
		}
	} else {
		// Remove socket if it exists (for non-jailer mode)
		os.Remove(process.SocketPath)
		cmd = exec.CommandContext(processCtx, "firecracker", "--api-sock", process.SocketPath)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start firecracker: %w", err)
	}

	process.Process = cmd.Process

	// Write PID file
	if err := os.WriteFile(process.PIDFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		m.logger.LogAttrs(ctx, slog.LevelWarn, "failed to write pid file",
			slog.String("error", err.Error()),
		)
	}

	// Wait for socket to become available
	if err := m.waitForSocket(ctx, process.SocketPath); err != nil {
		m.stopProcess(ctx, process)
		return nil, fmt.Errorf("firecracker failed to start: %w", err)
	}

	process.Status = StatusReady
	m.processes[processID] = process

	// Start monitoring process health in background
	go m.monitorProcess(process)

	m.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker process started successfully",
		slog.String("process_id", processID),
		slog.Int("pid", cmd.Process.Pid),
	)

	return process, nil
}

// waitForSocket waits for the Firecracker socket to become available
func (m *Manager) waitForSocket(ctx context.Context, socketPath string) error {
	timeout := time.After(m.startTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for socket %s", socketPath)
		case <-ticker.C:
			if _, err := os.Stat(socketPath); err == nil {
				// Socket exists, try to connect
				if err := m.testSocketConnection(socketPath); err == nil {
					return nil
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// testSocketConnection tests if we can connect to the socket
func (m *Manager) testSocketConnection(socketPath string) error {
	cmd := exec.Command("curl", "-s", "--unix-socket", socketPath, "http://localhost/")
	return cmd.Run()
}

// stopProcess stops a Firecracker process
func (m *Manager) stopProcess(ctx context.Context, proc *FirecrackerProcess) error {
	if proc.Status == StatusStopped || proc.Status == StatusStopping {
		return nil
	}

	proc.Status = StatusStopping

	m.logger.LogAttrs(ctx, slog.LevelInfo, "stopping firecracker process",
		slog.String("process_id", proc.ID),
		slog.Int("pid", proc.Process.Pid),
	)

	// Try graceful shutdown first
	if err := proc.Process.Signal(syscall.SIGTERM); err != nil {
		m.logger.LogAttrs(ctx, slog.LevelWarn, "failed to send SIGTERM",
			slog.String("process_id", proc.ID),
			slog.String("error", err.Error()),
		)
	}

	// Wait for graceful shutdown
	done := make(chan error, 1)
	go func() {
		_, err := proc.Process.Wait()
		done <- err
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill
		m.logger.LogAttrs(ctx, slog.LevelWarn, "force killing firecracker process",
			slog.String("process_id", proc.ID),
		)
		proc.Process.Kill()
		<-done // Wait for it to actually die
	case <-done:
		// Graceful shutdown completed
	}

	// Cleanup files
	os.Remove(proc.SocketPath)
	os.Remove(proc.PIDFile)

	// Cleanup jailer chroot directory if using jailer
	if proc.UseJailer && proc.ChrootPath != "" {
		if err := os.RemoveAll(proc.ChrootPath); err != nil {
			m.logger.LogAttrs(ctx, slog.LevelWarn, "failed to remove jailer chroot directory",
				slog.String("process_id", proc.ID),
				slog.String("chroot_path", proc.ChrootPath),
				slog.String("error", err.Error()),
			)
		} else {
			m.logger.LogAttrs(ctx, slog.LevelInfo, "cleaned up jailer chroot directory",
				slog.String("process_id", proc.ID),
				slog.String("chroot_path", proc.ChrootPath),
			)
		}
	}

	proc.Status = StatusStopped
	delete(m.processes, proc.ID)

	m.logger.LogAttrs(ctx, slog.LevelInfo, "firecracker process stopped",
		slog.String("process_id", proc.ID),
	)

	return nil
}

// cleanupLeftoverSockets removes any leftover socket files
func (m *Manager) cleanupLeftoverSockets() {
	files, err := filepath.Glob(filepath.Join(m.socketDir, "*.sock"))
	if err != nil {
		return
	}

	for _, file := range files {
		os.Remove(file)
	}
}

// GetProcessInfo returns information about managed processes
func (m *Manager) GetProcessInfo() map[string]*FirecrackerProcess {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	info := make(map[string]*FirecrackerProcess)
	for id, proc := range m.processes {
		// Create a copy to avoid race conditions
		info[id] = &FirecrackerProcess{
			ID:         proc.ID,
			SocketPath: proc.SocketPath,
			Started:    proc.Started,
			Status:     proc.Status,
			VMID:       proc.VMID,
		}
	}

	return info
}

// createProcessContext creates a context for Firecracker processes that:
// 1. Uses application context (long-lived) as base instead of request context (short-lived)
// 2. Copies tracing/observability data from request context for distributed tracing
// 3. Ensures processes survive beyond individual RPC requests
// AIDEV-NOTE: This solves the context lifecycle vs tracing dilemma
func (m *Manager) createProcessContext(requestCtx context.Context, processID string) context.Context {
	// Start with application context (long-lived)
	processCtx := m.appCtx

	// Copy observability data from request context for distributed tracing
	// This includes trace spans, baggage, and other observability metadata
	if requestCtx != nil {
		// Copy OpenTelemetry trace context for span correlation
		if span := trace.SpanFromContext(requestCtx); span.SpanContext().IsValid() {
			processCtx = trace.ContextWithSpan(processCtx, span)
		}

		// Copy baggage for multi-tenant context (tenant_id, user_id, etc.)
		// AIDEV-NOTE: Critical for multi-tenant systems - ensures tenant isolation in long-lived processes
		if requestBaggage := baggage.FromContext(requestCtx); len(requestBaggage.Members()) > 0 {
			processCtx = baggage.ContextWithBaggage(processCtx, requestBaggage)
		}

		// Note: We explicitly do NOT copy cancellation or deadlines from requestCtx
		// since we want processes to outlive individual requests
	}

	// Add process-specific metadata for observability
	processCtx = context.WithValue(processCtx, "process_id", processID)
	processCtx = context.WithValue(processCtx, "component", "firecracker_process")

	// Log tenant context for audit/security
	if requestBaggage := baggage.FromContext(processCtx); len(requestBaggage.Members()) > 0 {
		tenantID := requestBaggage.Member("tenant_id").Value()
		userID := requestBaggage.Member("user_id").Value()
		if tenantID != "" || userID != "" {
			m.logger.LogAttrs(context.Background(), slog.LevelInfo, "process created with tenant context",
				slog.String("process_id", processID),
				slog.String("tenant_id", tenantID),
				slog.String("user_id", userID),
			)
		}
	}

	return processCtx
}

// monitorProcess monitors a Firecracker process for unexpected exits
// AIDEV-NOTE: This goroutine detects when processes become defunct and cleans them up
func (m *Manager) monitorProcess(proc *FirecrackerProcess) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.LogAttrs(context.Background(), slog.LevelError, "process monitor panicked",
				slog.String("process_id", proc.ID),
				slog.Any("panic", r),
			)
		}
	}()

	// Wait for process to exit
	state, err := proc.Process.Wait()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Check if process still exists in our registry (might have been cleaned up already)
	if _, exists := m.processes[proc.ID]; !exists {
		return
	}

	if err != nil {
		m.logger.LogAttrs(context.Background(), slog.LevelError, "process wait failed",
			slog.String("process_id", proc.ID),
			slog.String("vm_id", proc.VMID),
			slog.String("error", err.Error()),
		)
		proc.Status = StatusError
	} else {
		exitCode := state.ExitCode()
		m.logger.LogAttrs(context.Background(), slog.LevelWarn, "firecracker process exited unexpectedly",
			slog.String("process_id", proc.ID),
			slog.String("vm_id", proc.VMID),
			slog.Int("exit_code", exitCode),
			slog.String("uptime", time.Since(proc.Started).String()),
		)
		proc.Status = StatusStopped
	}

	// Clean up socket and remove from registry
	os.Remove(proc.SocketPath)
	os.Remove(proc.PIDFile)
	delete(m.processes, proc.ID)

	m.logger.LogAttrs(context.Background(), slog.LevelInfo, "cleaned up defunct process",
		slog.String("process_id", proc.ID),
		slog.String("vm_id", proc.VMID),
	)
}

// createJailerCommand creates a jailer command with proper isolation settings
func (m *Manager) createJailerCommand(ctx context.Context, process *FirecrackerProcess) (*exec.Cmd, error) {
	if m.jailerConfig == nil || !m.jailerConfig.Enabled {
		return nil, fmt.Errorf("jailer not configured")
	}

	// Ensure chroot base directory exists
	if err := os.MkdirAll(m.jailerConfig.ChrootBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chroot base directory: %w", err)
	}

	// Build jailer arguments
	args := []string{
		"--id", process.JailerID,
		"--exec-file", m.jailerConfig.FirecrackerBinaryPath,
		"--uid", strconv.FormatUint(uint64(m.jailerConfig.UID), 10),
		"--gid", strconv.FormatUint(uint64(m.jailerConfig.GID), 10),
		"--chroot-base-dir", m.jailerConfig.ChrootBaseDir,
	}

	// Add namespace isolation flags
	if m.jailerConfig.NetNS {
		args = append(args, "--netns", "/var/run/netns/fc-"+process.JailerID)
	}

	// Add resource limits using cgroup v2 format
	args = append(args, "--cgroup-version", "2")
	
	if m.jailerConfig.ResourceLimits.MemoryLimitBytes > 0 {
		// Use cgroup v2 memory.max instead of v1 memory.limit_in_bytes
		args = append(args, "--resource-limit", fmt.Sprintf("memory.max=%d", m.jailerConfig.ResourceLimits.MemoryLimitBytes))
	}

	if m.jailerConfig.ResourceLimits.CPUQuota > 0 {
		// Use cgroup v2 cpu.max format: "quota period" (both in microseconds)
		// Period is typically 100ms = 100000 microseconds
		cpuQuotaUs := m.jailerConfig.ResourceLimits.CPUQuota * 1000 // Convert percentage to microseconds
		args = append(args, "--resource-limit", fmt.Sprintf("cpu.max=%d 100000", cpuQuotaUs))
	}

	// Add Firecracker arguments after --
	args = append(args, "--", "--api-sock", "/tmp/firecracker.socket")

	m.logger.LogAttrs(ctx, slog.LevelInfo, "creating jailer command",
		slog.String("jailer_binary", m.jailerConfig.BinaryPath),
		slog.String("jailer_id", process.JailerID),
		slog.String("chroot_path", process.ChrootPath),
		slog.Uint64("uid", uint64(m.jailerConfig.UID)),
		slog.Uint64("gid", uint64(m.jailerConfig.GID)),
		slog.Bool("netns", m.jailerConfig.NetNS),
		slog.Int64("memory_limit", m.jailerConfig.ResourceLimits.MemoryLimitBytes),
		slog.Int64("cpu_quota", m.jailerConfig.ResourceLimits.CPUQuota),
	)

	return exec.CommandContext(ctx, m.jailerConfig.BinaryPath, args...), nil
}
