# SQLC Query Analysis: Hydra Store Operations

This document analyzes all database operations currently implemented in the GORM store to understand what queries we need to implement in SQLC.

## Overview

The Hydra store has **25 distinct database operations** across 4 main entity types:
- **Workflow Executions** (11 operations)  
- **Workflow Steps** (4 operations)
- **Leases** (6 operations)
- **Cron Jobs** (4 operations)

## 1. Workflow Execution Operations

### 1.1 CreateWorkflow
**Purpose**: Insert a new workflow execution record  
**GORM Query**:
```go
s.db.WithContext(ctx).Create(workflow)
```
**SQL Equivalent**:
```sql
INSERT INTO workflow_executions (
    id, workflow_name, status, input_data, output_data, error_message,
    created_at, started_at, completed_at, max_attempts, remaining_attempts,
    next_retry_at, namespace, trigger_type, trigger_source, sleep_until,
    trace_id, span_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
```

### 1.2 GetWorkflow  
**Purpose**: Retrieve a specific workflow by ID and namespace  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("id = ? AND namespace = ?", id, namespace).
    First(&workflow)
```
**SQL Equivalent**:
```sql
SELECT * FROM workflow_executions 
WHERE id = ? AND namespace = ?
LIMIT 1
```

### 1.3 GetPendingWorkflows / GetPendingWorkflowsWithOffset
**Purpose**: Get workflows ready for execution (most critical query for performance)  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND (status = ? OR (status = ? AND next_retry_at <= ?) OR (status = ? AND sleep_until <= ?))",
        namespace,
        store.WorkflowStatusPending,
        store.WorkflowStatusFailed,
        now,
        store.WorkflowStatusSleeping,
        now,
    ).
    Where("workflow_name IN ?", workflowNames). // Optional filter
    Order("created_at ASC").
    Offset(offset).
    Limit(limit)
```
**SQL Equivalent**:
```sql
SELECT * FROM workflow_executions 
WHERE namespace = ? 
  AND (
    status = 'pending' 
    OR (status = 'failed' AND next_retry_at <= ?)
    OR (status = 'sleeping' AND sleep_until <= ?)
  )
  AND (? = 0 OR workflow_name = ANY(?))  -- Optional workflow name filter
ORDER BY created_at ASC 
LIMIT ? OFFSET ?
```
**Performance Critical**: This is the hottest query path - needs optimal indexing

### 1.4 UpdateWorkflowStatus
**Purpose**: Update workflow status and optionally error message  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Model(emptyWorkflowExecution).
    Where("id = ? AND namespace = ?", id, namespace).
    Updates(map[string]any{
        "status": status,
        "error_message": errorMsg,  // Optional
    })
```
**SQL Equivalent**:
```sql
UPDATE workflow_executions 
SET status = ?, error_message = COALESCE(?, error_message)
WHERE id = ? AND namespace = ?
```

### 1.5 CompleteWorkflow
**Purpose**: Mark workflow as completed with timestamp and optional output  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Model(emptyWorkflowExecution).
    Where("id = ? AND namespace = ?", id, namespace).
    Updates(map[string]any{
        "status": store.WorkflowStatusCompleted,
        "completed_at": now,
        "output_data": outputData,  // Optional
    })
```
**SQL Equivalent**:
```sql
UPDATE workflow_executions 
SET status = 'completed', 
    completed_at = ?, 
    output_data = COALESCE(?, output_data)
WHERE id = ? AND namespace = ?
```

### 1.6 FailWorkflow
**Purpose**: Handle workflow failure with retry logic (complex business logic)  
**GORM Operations**:
1. **Read current state**:
```go
s.db.WithContext(ctx).
    Where("id = ? AND namespace = ?", id, namespace).
    First(&workflow)
```
2. **Update with retry logic**:
```go
s.db.WithContext(ctx).
    Model(emptyWorkflowExecution).
    Where("id = ? AND namespace = ?", id, namespace).
    Updates(map[string]any{
        "error_message": errorMsg,
        "remaining_attempts": workflow.RemainingAttempts - 1,
        "status": store.WorkflowStatusFailed,
        "completed_at": now,           // If final failure
        "next_retry_at": nextRetry,    // If retryable
    })
```
**SQL Equivalent** (requires transaction):
```sql
-- First, get current state
SELECT id, max_attempts, remaining_attempts 
FROM workflow_executions 
WHERE id = ? AND namespace = ?;

-- Then update based on retry logic
UPDATE workflow_executions 
SET error_message = ?,
    remaining_attempts = remaining_attempts - 1,
    status = 'failed',
    completed_at = CASE WHEN (? = true OR remaining_attempts <= 1) THEN ? ELSE completed_at END,
    next_retry_at = CASE WHEN (? = false AND remaining_attempts > 1) THEN ? ELSE NULL END
WHERE id = ? AND namespace = ?
```

### 1.7 SleepWorkflow
**Purpose**: Put workflow to sleep until a specific time  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Model(emptyWorkflowExecution).
    Where("id = ? AND namespace = ?", id, namespace).
    Updates(map[string]any{
        "status": store.WorkflowStatusSleeping,
        "sleep_until": sleepUntil,
    })
```
**SQL Equivalent**:
```sql
UPDATE workflow_executions 
SET status = 'sleeping', sleep_until = ?
WHERE id = ? AND namespace = ?
```

### 1.8 GetSleepingWorkflows
**Purpose**: Find workflows ready to wake up from sleep  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND status = ? AND sleep_until <= ?",
        namespace, store.WorkflowStatusSleeping, beforeTime).
    Order("sleep_until ASC")
```
**SQL Equivalent**:
```sql
SELECT * FROM workflow_executions 
WHERE namespace = ? 
  AND status = 'sleeping' 
  AND sleep_until <= ?
ORDER BY sleep_until ASC
```

### 1.9 ResetOrphanedWorkflows  
**Purpose**: Reset running workflows that have no active lease (cleanup operation)  
**Raw SQL** (already optimized):
```sql
UPDATE workflow_executions 
SET status = 'pending' 
WHERE namespace = ? 
  AND status = 'running' 
  AND id NOT IN (
    SELECT resource_id 
    FROM leases 
    WHERE kind = 'workflow' AND namespace = ?
  )
```

### 1.10 GetAllWorkflows (Testing)
**Purpose**: Get all workflows in namespace for testing  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ?", namespace).
    Find(&workflows)
```
**SQL Equivalent**:
```sql
SELECT * FROM workflow_executions WHERE namespace = ?
```

## 2. Workflow Step Operations

### 2.1 CreateStep
**Purpose**: Insert a new workflow step record  
**GORM Query**:
```go
s.db.WithContext(ctx).Create(step)
```
**SQL Equivalent**:
```sql
INSERT INTO workflow_steps (
    id, execution_id, step_name, step_order, status, output_data,
    error_message, started_at, completed_at, max_attempts,
    remaining_attempts, namespace
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
```

### 2.2 GetStep
**Purpose**: Retrieve any step by execution, name  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND execution_id = ? AND step_name = ?",
        namespace, executionID, stepName).
    First(&step)
```
**SQL Equivalent**:
```sql
SELECT * FROM workflow_steps 
WHERE namespace = ? AND execution_id = ? AND step_name = ?
LIMIT 1
```

### 2.3 GetCompletedStep
**Purpose**: Retrieve only completed steps (for checkpointing)  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND execution_id = ? AND step_name = ? AND status = ?",
        namespace, executionID, stepName, store.StepStatusCompleted).
    First(&step)
```
**SQL Equivalent**:
```sql
SELECT * FROM workflow_steps 
WHERE namespace = ? 
  AND execution_id = ? 
  AND step_name = ? 
  AND status = 'completed'
LIMIT 1
```

### 2.4 UpdateStepStatus
**Purpose**: Update step status, output data, and completion time  
**GORM Operations**:
1. **Read current step**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND execution_id = ? AND step_name = ?", namespace, executionID, stepName).
    First(&step)
```
2. **Update step**:
```go
s.db.WithContext(ctx).
    Model(emptyWorkflowStep).
    Where("namespace = ? AND execution_id = ? AND step_name = ?", namespace, executionID, stepName).
    Updates(map[string]any{
        "status": status,
        "completed_at": now,
        "output_data": outputData,    // Optional
        "error_message": errorMsg,    // Optional
    })
```
**SQL Equivalent**:
```sql
UPDATE workflow_steps 
SET status = ?, 
    completed_at = ?,
    output_data = COALESCE(?, output_data),
    error_message = COALESCE(?, error_message)
WHERE namespace = ? AND execution_id = ? AND step_name = ?
```

### 2.5 GetAllSteps (Testing)
**Purpose**: Get all steps in namespace for testing  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ?", namespace).
    Find(&steps)
```
**SQL Equivalent**:
```sql
SELECT * FROM workflow_steps WHERE namespace = ?
```

## 3. Lease Operations

### 3.1 AcquireWorkflowLease
**Purpose**: Complex transactional lease acquisition (most critical for correctness)  
**GORM Transaction**: Multiple operations in sequence:

1. **Check workflow availability**:
```go
tx.Where("id = ? AND namespace = ?", workflowID, namespace).First(&workflow)
```

2. **Validate workflow state** (business logic in Go)

3. **Check existing lease**:
```go
tx.Where("resource_id = ? AND kind = ?", workflowID, "workflow").First(&existingLease)
```

4. **Create or update lease**:
```go
tx.Create(lease)  // For new lease
// OR
tx.Save(&existingLease)  // For renewal/takeover
```

5. **Update workflow to running**:
```go
tx.Model(emptyWorkflowExecution).
    Where("id = ? AND namespace = ?", workflowID, namespace).
    Updates(map[string]any{
        "status": store.WorkflowStatusRunning,
        "started_at": gorm.Expr("CASE WHEN started_at IS NULL THEN ? ELSE started_at END", now),
        "sleep_until": nil,
    })
```

**SQL Equivalent** (single transaction with proper locking):
```sql
BEGIN;

-- 1. Lock and validate workflow
SELECT id, status, next_retry_at, sleep_until 
FROM workflow_executions 
WHERE id = ? AND namespace = ?
FOR UPDATE;

-- 2. Check/acquire lease with proper upsert
INSERT INTO leases (resource_id, kind, namespace, worker_id, acquired_at, expires_at, heartbeat_at)
VALUES (?, 'workflow', ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  worker_id = CASE 
    WHEN expires_at <= ? THEN VALUES(worker_id)  -- Expired, can take over
    WHEN worker_id = VALUES(worker_id) THEN VALUES(worker_id)  -- Same worker, can renew
    ELSE worker_id  -- Different worker, keep existing
  END,
  acquired_at = CASE 
    WHEN expires_at <= ? OR worker_id = VALUES(worker_id) THEN VALUES(acquired_at)
    ELSE acquired_at 
  END,
  expires_at = CASE 
    WHEN expires_at <= ? OR worker_id = VALUES(worker_id) THEN VALUES(expires_at)
    ELSE expires_at 
  END,
  heartbeat_at = CASE 
    WHEN expires_at <= ? OR worker_id = VALUES(worker_id) THEN VALUES(heartbeat_at)
    ELSE heartbeat_at 
  END;

-- 3. Update workflow status if lease was acquired
UPDATE workflow_executions 
SET status = 'running',
    started_at = CASE WHEN started_at IS NULL THEN ? ELSE started_at END,
    sleep_until = NULL
WHERE id = ? AND namespace = ?
  AND EXISTS (
    SELECT 1 FROM leases 
    WHERE resource_id = ? AND worker_id = ?
  );

COMMIT;
```

### 3.2 AcquireLease (Generic)
**Purpose**: Simple lease creation for cron jobs  
**GORM Query**:
```go
s.db.WithContext(ctx).Create(lease)
```
**SQL Equivalent**:
```sql
INSERT INTO leases (resource_id, kind, namespace, worker_id, acquired_at, expires_at, heartbeat_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
```

### 3.3 HeartbeatLease
**Purpose**: Update lease expiration and heartbeat time  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Model(emptyLease).
    Where("resource_id = ? AND worker_id = ?", resourceID, workerID).
    Updates(map[string]any{
        "heartbeat_at": now,
        "expires_at": expiresAt,
    })
```
**SQL Equivalent**:
```sql
UPDATE leases 
SET heartbeat_at = ?, expires_at = ?
WHERE resource_id = ? AND worker_id = ?
```

### 3.4 ReleaseLease
**Purpose**: Delete a lease owned by specific worker  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("resource_id = ? AND worker_id = ?", resourceID, workerID).
    Delete(emptyLease)
```
**SQL Equivalent**:
```sql
DELETE FROM leases 
WHERE resource_id = ? AND worker_id = ?
```

### 3.5 GetLease
**Purpose**: Retrieve lease information by resource ID  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("resource_id = ?", resourceID).
    First(&lease)
```
**SQL Equivalent**:
```sql
SELECT * FROM leases WHERE resource_id = ? LIMIT 1
```

### 3.6 CleanupExpiredLeases
**Purpose**: Bulk delete expired leases  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND expires_at < ?", namespace, now).
    Delete(emptyLease)
```
**SQL Equivalent**:
```sql
DELETE FROM leases 
WHERE namespace = ? AND expires_at < ?
```

### 3.7 GetExpiredLeases
**Purpose**: Get list of expired leases (for monitoring)  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND expires_at < ?", namespace, now).
    Find(&leases)
```
**SQL Equivalent**:
```sql
SELECT * FROM leases 
WHERE namespace = ? AND expires_at < ?
```

## 4. Cron Job Operations

### 4.1 UpsertCronJob
**Purpose**: Insert or update cron job configuration  
**GORM Operations**:
1. **Check if exists**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND name = ?", cronJob.Namespace, cronJob.Name).
    First(&existing)
```
2. **Update or create**:
```go
s.db.WithContext(ctx).Save(cronJob)  // If exists
// OR
s.db.WithContext(ctx).Create(cronJob)  // If new
```
**SQL Equivalent**:
```sql
INSERT INTO cron_jobs (id, name, cron_spec, namespace, workflow_name, enabled, created_at, updated_at, last_run_at, next_run_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  cron_spec = VALUES(cron_spec),
  workflow_name = VALUES(workflow_name),
  enabled = VALUES(enabled),
  updated_at = VALUES(updated_at),
  next_run_at = VALUES(next_run_at)
```

### 4.2 GetCronJob
**Purpose**: Retrieve specific cron job by namespace and name  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND name = ?", namespace, name).
    First(&cronJob)
```
**SQL Equivalent**:
```sql
SELECT * FROM cron_jobs 
WHERE namespace = ? AND name = ?
LIMIT 1
```

### 4.3 GetCronJobs
**Purpose**: Get all enabled cron jobs in namespace  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND enabled = ?", namespace, true).
    Find(&cronJobs)
```
**SQL Equivalent**:
```sql
SELECT * FROM cron_jobs 
WHERE namespace = ? AND enabled = true
```

### 4.4 GetDueCronJobs
**Purpose**: Get cron jobs ready to execute  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Where("namespace = ? AND enabled = ? AND next_run_at <= ?", namespace, true, beforeTime).
    Find(&cronJobs)
```
**SQL Equivalent**:
```sql
SELECT * FROM cron_jobs 
WHERE namespace = ? 
  AND enabled = true 
  AND next_run_at <= ?
```

### 4.5 UpdateCronJobLastRun
**Purpose**: Update cron job execution timestamps  
**GORM Query**:
```go
s.db.WithContext(ctx).
    Model(emptyCronJob).
    Where("id = ? AND namespace = ?", cronJobID, namespace).
    Updates(map[string]any{
        "last_run_at": lastRunAt,
        "next_run_at": nextRunAt,
        "updated_at": time.Now().UnixMilli(),
    })
```
**SQL Equivalent**:
```sql
UPDATE cron_jobs 
SET last_run_at = ?, next_run_at = ?, updated_at = ?
WHERE id = ? AND namespace = ?
```

## 5. Transaction Support

### WithTx
**Purpose**: Execute multiple operations in a transaction  
**GORM Implementation**:
```go
s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    txStore := &gormStore{db: tx, clock: s.clock}
    return fn(txStore)
})
```
**SQLC Implementation**: Will use `database/sql` transactions with `tx.Begin()` / `tx.Commit()` / `tx.Rollback()`

## 6. Critical Performance & Security Considerations

### 6.1 Hot Path Queries (Need Optimal Indexing)
1. **GetPendingWorkflows** - Most frequent query, needs composite index on (namespace, status, created_at)
2. **AcquireWorkflowLease** - Race condition critical, needs proper SELECT FOR UPDATE
3. **HeartbeatLease** - High frequency, needs index on (resource_id, worker_id)

### 6.2 Race Condition Fixes Needed
1. **AcquireWorkflowLease** - Current GORM implementation has TOCTOU races
2. **Step creation/updates** - Need better concurrency control
3. **Lease operations** - Need atomic upsert patterns

### 6.3 Missing in Current Implementation
1. **Proper row locking** with SELECT FOR UPDATE
2. **Atomic upsert** operations for lease acquisition  
3. **Bulk operations** for better performance
4. **Query timeouts** and resource limits

## 7. SQLC Query Organization

Suggested file structure for SQLC queries:
```
store/sqlc/queries/
├── workflows.sql       # Workflow execution operations (1.1-1.10)
├── steps.sql          # Workflow step operations (2.1-2.5)  
├── leases.sql         # Lease operations (3.1-3.7)
├── cron.sql           # Cron job operations (4.1-4.5)
└── transactions.sql   # Complex transactional operations
```

Each file will contain the corresponding SQL queries with proper parameter binding and return types for SQLC code generation.