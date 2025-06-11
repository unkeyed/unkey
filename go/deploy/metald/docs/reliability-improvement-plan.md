# Metald Reliability Improvement Plan

## Executive Summary

This plan addresses critical reliability gaps in metald's VM process management that could impact customer workloads. The current system lacks robust orphaned VM detection, process recovery, and state reconciliation capabilities.

## Current Reliability Gaps

### 1. Orphaned VM State Issues
- **Problem**: VMs can become orphaned when Firecracker processes die unexpectedly
- **Impact**: Customer VMs appear "running" but are unreachable
- **Root Cause**: Registry state != actual process state

### 2. No Health Checking
- **Problem**: No proactive detection of unhealthy VMs
- **Impact**: Silent failures until customer attempts to use VM
- **Root Cause**: Missing health check mechanisms

### 3. Limited Recovery Capabilities  
- **Problem**: No automatic recovery from process failures
- **Impact**: Manual intervention required for failed VMs
- **Root Cause**: No process restart/recovery logic

### 4. Inconsistent State Management
- **Problem**: VM registry can drift from actual process state
- **Impact**: API operations may fail unexpectedly
- **Root Cause**: No reconciliation between registry and reality

## Implementation Plan

### Phase 1: Foundation (High Priority)

#### 1.1 VM Health Checking System
```go
// Add to managed_client.go
type VMHealthStatus struct {
    VMId        string
    IsHealthy   bool
    LastCheck   time.Time
    ProcessPID  int
    SocketPath  string
    ErrorMsg    string
}

func (mc *ManagedClient) CheckVMHealth(vmID string) *VMHealthStatus
func (mc *ManagedClient) StartHealthChecker() // Background goroutine
```

**Benefits**: Early detection of failed VMs, proactive alerting

#### 1.2 Orphaned VM Detection
```go
// Add to manager.go  
type OrphanedVM struct {
    VMId       string
    ProcessID  string
    LastSeen   time.Time
    Reason     string
}

func (m *Manager) DetectOrphanedVMs() []*OrphanedVM
func (m *Manager) RecoverOrphanedVM(vmID string) error
```

**Benefits**: Automatic detection and recovery of orphaned VMs

#### 1.3 Process Recovery System
```go
// Add to manager.go
type RecoveryConfig struct {
    MaxRetries    int
    RetryInterval time.Duration
    BackoffFactor float64
}

func (m *Manager) RecoverProcess(vmID string, config *RecoveryConfig) error
func (m *Manager) RestartProcess(processID string) error
```

**Benefits**: Automatic recovery from transient failures

### Phase 2: State Reconciliation (Medium Priority)

#### 2.1 Reconciliation Loop
```go
// New file: internal/reconciler/reconciler.go
type Reconciler struct {
    managedClient *ManagedClient
    interval      time.Duration
    logger        *slog.Logger
}

func (r *Reconciler) ReconcileVMState() error
func (r *Reconciler) StartReconcileLoop() // Background process
```

**Benefits**: Ensures registry matches reality, prevents state drift

#### 2.2 Enhanced VM State Tracking
```go
// Extend managedVM struct
type managedVM struct {
    ID           string
    ProcessID    string
    Config       *metaldv1.VmConfig
    State        metaldv1.VmState
    Client       *Client
    HealthStatus *VMHealthStatus  // NEW
    LastActivity time.Time        // NEW
    RecoveryCount int             // NEW
}
```

**Benefits**: Better visibility into VM lifecycle and health

### Phase 3: Monitoring & Alerting (Medium Priority)

#### 3.1 Enhanced Metrics
```go
// Add to observability/metrics.go
vmOrphanedTotal        metric.Int64Counter
vmRecoveryAttempts     metric.Int64Counter  
vmRecoverySuccess      metric.Int64Counter
vmHealthCheckFailures  metric.Int64Counter
processRestartTotal    metric.Int64Counter
```

**Benefits**: Operational visibility into failures and recovery

#### 3.2 Alerting Integration via Log-based Monitoring

**Benefits**: Use existing log aggregation and alerting infrastructure

**Implementation**: Configure log-based alerts on structured log messages:
- `"orphaned vm detected"` - VM orphan alerts
- `"vm recovery.*failed completely"` - Recovery failure alerts
- `"vm health check failed"` - Health check failure alerts

### Phase 4: Advanced Features (Low Priority)

#### 4.1 Graceful Degradation
```go
// Add to managed_client.go
type DegradationMode string

const (
    ModeNormal     DegradationMode = "normal"
    ModeReadOnly   DegradationMode = "readonly"  
    ModeMaintenance DegradationMode = "maintenance"
)

func (mc *ManagedClient) SetDegradationMode(mode DegradationMode)
```

**Benefits**: Maintain service availability during issues

#### 4.2 VM Migration Support
```go
// Future: VM migration between processes
func (mc *ManagedClient) MigrateVM(vmID, newProcessID string) error
```

**Benefits**: Zero-downtime recovery from process failures

## Implementation Details

### Health Check Implementation

```go
// internal/health/vm_health.go
package health

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "time"
)

type VMHealthChecker struct {
    client     *http.Client
    timeout    time.Duration
    interval   time.Duration
    logger     *slog.Logger
}

func (hc *VMHealthChecker) CheckVM(socketPath string) error {
    // 1. Check socket exists and is accessible
    if _, err := net.Dial("unix", socketPath); err != nil {
        return fmt.Errorf("socket unreachable: %w", err)
    }
    
    // 2. Check Firecracker API responds
    resp, err := hc.client.Get("http://unix/")
    if err != nil {
        return fmt.Errorf("api check failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("api returned error: %d", resp.StatusCode)
    }
    
    return nil
}

func (hc *VMHealthChecker) StartPeriodicCheck(vmID, socketPath string, onFailure func(error)) {
    ticker := time.NewTicker(hc.interval)
    go func() {
        defer ticker.Stop()
        for range ticker.C {
            if err := hc.CheckVM(socketPath); err != nil {
                hc.logger.Error("vm health check failed",
                    "vm_id", vmID,
                    "error", err,
                )
                onFailure(err)
            }
        }
    }()
}
```

### Orphaned VM Detection

```go
// Add to process/manager.go
func (m *Manager) DetectOrphanedVMs() map[string]*OrphanedVM {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    orphaned := make(map[string]*OrphanedVM)
    
    for processID, proc := range m.processes {
        if proc.VMID == "" {
            continue // Process not assigned to VM
        }
        
        // Check if process is actually running
        if !m.isProcessRunning(proc.Process.Pid) {
            orphaned[proc.VMID] = &OrphanedVM{
                VMId:      proc.VMID,
                ProcessID: processID,
                LastSeen:  proc.Started,
                Reason:    "process_dead",
            }
            continue
        }
        
        // Check if socket is accessible
        if err := m.testSocketConnection(proc.SocketPath); err != nil {
            orphaned[proc.VMID] = &OrphanedVM{
                VMId:      proc.VMID, 
                ProcessID: processID,
                LastSeen:  time.Now(),
                Reason:    "socket_unreachable",
            }
        }
    }
    
    return orphaned
}

func (m *Manager) isProcessRunning(pid int) bool {
    process, err := os.FindProcess(pid)
    if err != nil {
        return false
    }
    
    // Send signal 0 to check if process exists
    err = process.Signal(syscall.Signal(0))
    return err == nil
}
```

### Recovery System

```go
// Add to managed_client.go
func (mc *ManagedClient) RecoverVM(ctx context.Context, vmID string) error {
    mc.logger.Info("attempting vm recovery", "vm_id", vmID)
    
    managedVm, exists := mc.vmRegistry[vmID]
    if !exists {
        return fmt.Errorf("vm %s not in registry", vmID)
    }
    
    // 1. Try to reconnect to existing process
    if err := mc.reconnectToProcess(ctx, managedVm); err == nil {
        mc.logger.Info("vm recovered via reconnection", "vm_id", vmID)
        return nil
    }
    
    // 2. Create new process and restore VM state
    newProcess, err := mc.processManager.GetOrCreateProcess(ctx, vmID)
    if err != nil {
        return fmt.Errorf("failed to create recovery process: %w", err)
    }
    
    // 3. Recreate VM on new process
    newClient := mc.createClientForProcess(newProcess)
    _, err = newClient.CreateVMWithID(ctx, managedVm.Config, vmID)
    if err != nil {
        mc.processManager.ReleaseProcess(ctx, vmID)
        return fmt.Errorf("failed to recreate vm: %w", err)
    }
    
    // 4. Update registry
    managedVm.Client = newClient
    managedVm.ProcessID = newProcess.ID
    
    mc.logger.Info("vm recovered successfully", "vm_id", vmID)
    return nil
}
```

## Testing Strategy

### Unit Tests
- Health check logic
- Orphaned VM detection  
- Recovery mechanisms
- State reconciliation

### Integration Tests
- Process failure simulation
- Socket removal scenarios
- End-to-end recovery flows
- Concurrent recovery attempts

### Chaos Testing
- Random process kills
- Socket corruption
- Resource exhaustion
- Network partitions

## Monitoring & Metrics

### Key Metrics to Track
```
metald_vm_orphaned_total{reason}           - Orphaned VM count by reason
metald_vm_recovery_attempts_total{outcome} - Recovery attempts and outcomes  
metald_vm_health_check_failures_total      - Health check failure count
metald_process_restart_total{reason}       - Process restart count by reason
metald_reconcile_duration_seconds          - Reconciliation loop timing
metald_vm_downtime_seconds                 - VM unavailability duration
```

### Alerts to Configure
- VM orphaned for >5 minutes
- Recovery failure rate >10%
- Health check failure rate >5%
- Reconciliation loop stalled

## Migration Strategy

### Phase 1 (Weeks 1-2)
1. Implement VM health checking
2. Add orphaned VM detection
3. Basic process recovery

### Phase 2 (Weeks 3-4)  
1. State reconciliation loop
2. Enhanced metrics and monitoring
3. Integration testing

### Phase 3 (Weeks 5-6)
1. Alerting integration
2. Graceful degradation
3. Chaos testing and validation

### Rollout Plan
1. **Development**: Feature flags for gradual enablement
2. **Staging**: Full validation with customer-like workloads  
3. **Production**: Phased rollout with monitoring

## Risk Mitigation

### Backwards Compatibility
- All new features behind feature flags
- Graceful fallback to current behavior
- Zero breaking changes to existing APIs

### Performance Impact
- Health checks run in background goroutines
- Configurable check intervals (default: 30s)
- Minimal overhead on hot path operations

### Operational Safety
- Manual override for recovery operations
- Circuit breakers for automatic recovery
- Comprehensive logging for troubleshooting

## Success Criteria

### Reliability Metrics
- VM orphan rate: <0.1% of total VMs
- Recovery success rate: >95%
- Mean time to recovery: <30 seconds
- False positive rate: <1%

### Operational Metrics  
- Manual intervention events: Reduce by 80%
- Customer support tickets: Reduce by 60%
- Mean time to detection: <1 minute

## Conclusion

This reliability improvement plan addresses critical gaps in metald's VM management that could impact customer workloads. The phased approach ensures minimal risk while delivering significant reliability improvements.

The investment in robust health checking, orphaned VM recovery, and state reconciliation will provide a foundation for reliable multi-tenant VM management at scale.