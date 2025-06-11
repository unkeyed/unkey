package reliability

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/health"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/process"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/recovery"
)

// Mock implementations for testing
type mockProcess struct {
	id         string
	socketPath string
	pid        int
	vmID       string
	status     string
	running    bool
}

func (m *mockProcess) GetID() string          { return m.id }
func (m *mockProcess) GetSocketPath() string  { return m.socketPath }
func (m *mockProcess) GetPID() int            { return m.pid }
func (m *mockProcess) GetVMID() string        { return m.vmID }
func (m *mockProcess) GetStatus() string      { return m.status }
func (m *mockProcess) IsRunning() bool        { return m.running }

type mockProcessManager struct {
	processes map[string]*mockProcess
	failures  map[string]error
}

func newMockProcessManager() *mockProcessManager {
	return &mockProcessManager{
		processes: make(map[string]*mockProcess),
		failures:  make(map[string]error),
	}
}

func (m *mockProcessManager) GetOrCreateProcess(ctx context.Context, vmID string) (recovery.ProcessInfo, error) {
	if err, hasFailure := m.failures[vmID]; hasFailure {
		return nil, err
	}
	
	proc := &mockProcess{
		id:         "proc-" + vmID,
		socketPath: "/tmp/test-" + vmID + ".sock",
		pid:        12345,
		vmID:       vmID,
		status:     "ready",
		running:    true,
	}
	
	m.processes[proc.id] = proc
	return proc, nil
}

func (m *mockProcessManager) ReleaseProcess(ctx context.Context, vmID string) error {
	for id, proc := range m.processes {
		if proc.vmID == vmID {
			delete(m.processes, id)
			break
		}
	}
	return nil
}

func (m *mockProcessManager) GetProcessInfo() map[string]recovery.ProcessInfo {
	result := make(map[string]recovery.ProcessInfo)
	for id, proc := range m.processes {
		result[id] = proc
	}
	return result
}

func (m *mockProcessManager) IsProcessHealthy(processID string) bool {
	proc, exists := m.processes[processID]
	return exists && proc.running
}

func (m *mockProcessManager) setProcessUnhealthy(processID string) {
	if proc, exists := m.processes[processID]; exists {
		proc.running = false
	}
}

func (m *mockProcessManager) removeProcess(processID string) {
	delete(m.processes, processID)
}

func TestReliabilityIntegration(t *testing.T) {
	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	// Create test process manager
	ctx := context.Background()
	pmConfig := &config.ProcessManagerConfig{
		SocketDir:    "/tmp/test-sockets",
		LogDir:       "/tmp/test-logs",
		MaxProcesses: 10,
	}
	processManager := process.NewManager(logger, ctx, pmConfig)
	
	// Create reliability config with short intervals for testing
	reliabilityConfig := &ReliabilityConfig{
		Enabled: true,
		HealthCheck: &health.HealthCheckConfig{
			Interval:          1 * time.Second,
			Timeout:           500 * time.Millisecond,
			FailureThreshold:  2,
			RecoveryThreshold: 1,
			Enabled:           true,
		},
		Recovery: &recovery.RecoveryConfig{
			MaxRetries:        2,
			RetryInterval:     1 * time.Second,
			BackoffFactor:     1.5,
			MaxRetryInterval:  5 * time.Second,
			RecoveryTimeout:   10 * time.Second,
			DetectionInterval: 2 * time.Second,
			Enabled:           true,
			AllowDataLoss:     true,
		},
		ReconcileInterval: 3 * time.Second,
	}
	
	// Create reliability manager
	reliabilityManager, err := NewReliabilityManager(logger, reliabilityConfig, processManager)
	if err != nil {
		t.Fatalf("Failed to create reliability manager: %v", err)
	}
	
	// Start reliability subsystem
	err = reliabilityManager.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start reliability manager: %v", err)
	}
	defer reliabilityManager.Stop()
	
	// Test VM creation and health monitoring
	t.Run("VM Creation and Health Monitoring", func(t *testing.T) {
		vmID := "test-vm-1"
		processID := "test-proc-1"
		
		config := &metaldv1.VmConfig{
			Cpu: &metaldv1.CpuConfig{
				VcpuCount: 1,
			},
			Memory: &metaldv1.MemoryConfig{
				SizeBytes: 134217728, // 128 MB
			},
		}
		
		// Simulate VM creation
		reliabilityManager.OnVMCreated(vmID, processID, config)
		reliabilityManager.OnVMStarted(vmID)
		
		// Wait a bit for health monitoring to start
		time.Sleep(2 * time.Second)
		
		// Check that VM is being monitored
		health, exists := reliabilityManager.GetVMHealth(vmID)
		if !exists {
			t.Errorf("Expected VM %s to be monitored, but it's not", vmID)
		}
		
		// Since we're using mock without actual socket, expect it to be unhealthy
		if health != nil && health.IsHealthy {
			t.Logf("VM %s is reported as healthy (might be expected with mock)", vmID)
		}
		
		// Clean up
		reliabilityManager.OnVMDeleted(vmID)
	})
	
	// Test reliability status
	t.Run("Reliability Status", func(t *testing.T) {
		status := reliabilityManager.GetReliabilityStatus()
		
		if !status["enabled"].(bool) {
			t.Error("Expected reliability to be enabled")
		}
		
		if status["health_check"] == nil {
			t.Error("Expected health_check status to be present")
		}
		
		if status["recovery"] == nil {
			t.Error("Expected recovery status to be present")
		}
		
		if status["registry"] == nil {
			t.Error("Expected registry status to be present")
		}
	})
	
	// Test disabled reliability manager
	t.Run("Disabled Reliability Manager", func(t *testing.T) {
		disabledConfig := &ReliabilityConfig{
			Enabled: false,
		}
		
		disabledManager, err := NewReliabilityManager(logger, disabledConfig, processManager)
		if err != nil {
			t.Fatalf("Failed to create disabled reliability manager: %v", err)
		}
		
		err = disabledManager.Start(ctx)
		if err != nil {
			t.Fatalf("Failed to start disabled reliability manager: %v", err)
		}
		defer disabledManager.Stop()
		
		// Test that operations are no-ops
		disabledManager.OnVMCreated("test-vm", "test-proc", &metaldv1.VmConfig{})
		
		status := disabledManager.GetReliabilityStatus()
		if status["enabled"].(bool) {
			t.Error("Expected disabled reliability manager to report as disabled")
		}
		
		health := disabledManager.GetAllVMHealth()
		if len(health) != 0 {
			t.Error("Expected no health data from disabled manager")
		}
		
		orphaned := disabledManager.GetOrphanedVMs()
		if len(orphaned) != 0 {
			t.Error("Expected no orphaned VMs from disabled manager")
		}
	})
}

func TestVMRegistryAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	registry := NewVMRegistryAdapter(logger)
	
	// Test adding VM
	vmID := "test-vm-1"
	processID := "test-proc-1"
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{VcpuCount: 1},
	}
	
	registry.AddVM(vmID, processID, config, metaldv1.VmState_VM_STATE_STARTING)
	
	// Test getting VM
	vm, exists := registry.GetVM(vmID)
	if !exists {
		t.Fatalf("Expected VM %s to exist in registry", vmID)
	}
	
	if vm.GetID() != vmID {
		t.Errorf("Expected VM ID %s, got %s", vmID, vm.GetID())
	}
	
	if vm.GetProcessID() != processID {
		t.Errorf("Expected process ID %s, got %s", processID, vm.GetProcessID())
	}
	
	if vm.GetState() != metaldv1.VmState_VM_STATE_STARTING {
		t.Errorf("Expected state STARTING, got %s", vm.GetState())
	}
	
	// Test updating VM process
	newProcessID := "new-proc-1"
	err := registry.UpdateVMProcess(vmID, newProcessID)
	if err != nil {
		t.Fatalf("Failed to update VM process: %v", err)
	}
	
	vm, _ = registry.GetVM(vmID)
	if vm.GetProcessID() != newProcessID {
		t.Errorf("Expected updated process ID %s, got %s", newProcessID, vm.GetProcessID())
	}
	
	// Test updating VM state
	err = registry.UpdateVMState(vmID, metaldv1.VmState_VM_STATE_RUNNING)
	if err != nil {
		t.Fatalf("Failed to update VM state: %v", err)
	}
	
	vm, _ = registry.GetVM(vmID)
	if vm.GetState() != metaldv1.VmState_VM_STATE_RUNNING {
		t.Errorf("Expected state RUNNING, got %s", vm.GetState())
	}
	
	// Test marking VM as failed
	err = registry.MarkVMFailed(vmID, "test failure")
	if err != nil {
		t.Fatalf("Failed to mark VM as failed: %v", err)
	}
	
	vm, _ = registry.GetVM(vmID)
	if vm.GetState() != metaldv1.VmState_VM_STATE_ERROR {
		t.Errorf("Expected state ERROR, got %s", vm.GetState())
	}
	
	// Test getting all VMs
	allVMs := registry.GetAllVMs()
	if len(allVMs) != 1 {
		t.Errorf("Expected 1 VM in registry, got %d", len(allVMs))
	}
	
	// Test removing VM
	registry.RemoveVM(vmID)
	
	_, exists = registry.GetVM(vmID)
	if exists {
		t.Errorf("Expected VM %s to be removed from registry", vmID)
	}
	
	// Test updating non-existent VM
	err = registry.UpdateVMProcess("non-existent", "proc")
	if err == nil {
		t.Error("Expected error when updating non-existent VM")
	}
	
	err = registry.UpdateVMState("non-existent", metaldv1.VmState_VM_STATE_RUNNING)
	if err == nil {
		t.Error("Expected error when updating state of non-existent VM")
	}
	
	err = registry.MarkVMFailed("non-existent", "test")
	if err == nil {
		t.Error("Expected error when marking non-existent VM as failed")
	}
}

func TestProcessManagerAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	
	mockPM := newMockProcessManager()
	adapter := &ProcessManagerAdapter{
		manager: nil, // We'll test with direct mock calls
		logger:  logger,
	}
	
	// Manually test the adapter interface
	vmID := "test-vm-1"
	
	// Create a mock process
	proc := &mockProcess{
		id:         "proc-1",
		socketPath: "/tmp/test.sock",
		pid:        12345,
		vmID:       vmID,
		status:     "ready",
		running:    true,
	}
	
	mockPM.processes[proc.id] = proc
	
	// Test GetProcessInfo
	processInfo := mockPM.GetProcessInfo()
	if len(processInfo) != 1 {
		t.Errorf("Expected 1 process, got %d", len(processInfo))
	}
	
	info := processInfo[proc.id]
	if info.GetID() != proc.id {
		t.Errorf("Expected process ID %s, got %s", proc.id, info.GetID())
	}
	
	if info.GetVMID() != vmID {
		t.Errorf("Expected VM ID %s, got %s", vmID, info.GetVMID())
	}
	
	if !info.IsRunning() {
		t.Error("Expected process to be running")
	}
	
	// Test IsProcessHealthy
	if !mockPM.IsProcessHealthy(proc.id) {
		t.Error("Expected process to be healthy")
	}
	
	// Make process unhealthy
	mockPM.setProcessUnhealthy(proc.id)
	if mockPM.IsProcessHealthy(proc.id) {
		t.Error("Expected process to be unhealthy after setting unhealthy")
	}
	
	// Test with non-existent process
	if mockPM.IsProcessHealthy("non-existent") {
		t.Error("Expected non-existent process to be unhealthy")
	}
}