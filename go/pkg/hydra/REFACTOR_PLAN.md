# File Organization Refactor Plan

## Current State Analysis

### Current Files:
- `hydra.go` - Mixed: Engine, Worker, Workflow registration
- `worker.go` - Worker implementation 
- `step.go` - Step execution logic
- `types.go` - All type definitions
- `models.go` - Type aliases
- `hydra_test.go` - Tests
- `test_helpers.go` - Test utilities
- `testharness/` - Event collector for tests

## Target File Organization

### `engine.go` - Engine core functionality
**Structs:**
- [ ] `Engine` struct (from hydra.go)
- [ ] `Config` struct (from hydra.go)

**Functions:**
- [ ] `New(config Config) *Engine` (from hydra.go)
- [ ] `NewWithStore(...) *Engine` (from hydra.go)
- [ ] `(e *Engine) StartWorkflow(...)` (from hydra.go)
- [ ] `(e *Engine) RegisterCron(...)` (from hydra.go)
- [ ] `(e *Engine) GetStore()` (from hydra.go)
- [ ] `generateWorkerID()` (from hydra.go) - utility function

### `worker.go` - Worker functionality
**Structs:**
- [x] `Worker` interface
- [x] `worker` struct (already in worker.go)
- [x] `WorkerConfig` struct (from hydra.go)

**Functions:**
- [x] `NewWorker(...)` (already in worker.go)
- [x] `(w *worker) Start(...)` (already in worker.go)
- [x] `(w *worker) Shutdown(...)` (already in worker.go)
- [x] `(w *worker) run(...)` (already in worker.go)
- [x] `(w *worker) pollForWorkflows(...)` (already in worker.go)
- [x] `(w *worker) pollOnce(...)` (already in worker.go)
- [x] `(w *worker) executeWorkflow(...)` (already in worker.go)
- [x] `(w *worker) sendHeartbeats(...)` (already in worker.go)
- [x] `(w *worker) cleanupExpiredLeases(...)` (already in worker.go)
- [x] `(w *worker) processCronJobs(...)` (already in worker.go)
- [x] `(w *worker) processDueCronJobs(...)` (already in worker.go)
- [x] `(w *worker) executeCronJob(...)` (already in worker.go)

### `workflow.go` - Workflow types and registration
**Structs:**
- [ ] `Workflow[TReq]` interface (from types.go)
- [ ] `GenericWorkflow` type alias (from types.go)
- [ ] `WorkflowContext` interface (from types.go)
- [ ] `workflowContext` struct (from types.go)
- [ ] `workflowWrapper[TReq]` struct (from hydra.go)
- [ ] `WorkflowConfig` struct (from types.go)
- [ ] `WorkflowSuspendedError` struct (from types.go)

**Functions:**
- [ ] `RegisterWorkflow[TReq](...)` (from hydra.go)
- [ ] `(w *workflowWrapper[TReq]) Name()` (from hydra.go)
- [ ] `(w *workflowWrapper[TReq]) Run(...)` (from hydra.go)
- [ ] `(w *workflowContext) Context()` (from types.go)
- [ ] `(w *workflowContext) ExecutionID()` (from types.go)
- [ ] `(w *workflowContext) WorkflowName()` (from types.go)
- [ ] `(w *workflowContext) getNextStepOrder()` (from types.go)
- [ ] `(w *workflowContext) getCompletedStep(...)` (from types.go)
- [ ] `(w *workflowContext) getAnyStep(...)` (from types.go)
- [ ] `(w *workflowContext) markStepCompleted(...)` (from types.go)
- [ ] `(w *workflowContext) markStepFailed(...)` (from types.go)
- [ ] `(w *workflowContext) suspendWorkflowForSleep(...)` (from types.go)
- [ ] Workflow option functions: `WithMaxAttempts`, `WithTimeout`, `WithRetryBackoff` (from types.go)
- [ ] `(e *WorkflowSuspendedError) Error()` (from types.go)

### `step.go` - Step execution
**Functions:**
- [x] `Step[TResponse](...)` (step execution logic)

### `sleep.go` - Sleep functionality (new file)
**Functions:**
- [x] `Sleep(...)` (workflow sleep/suspension logic)

### `types.go` - REMOVED ❌
**Reason:** All types moved to appropriate domain files:
- `Payload` - Replaced with `any` in `engine.go` 
- `WorkflowHandler` - Unused legacy type, removed
- `Worker` interface - Moved to `worker.go`
- Core types now live in `models.go` and domain-specific files

### `cron.go` - Cron-related functionality (new file)
**Types:**
- [ ] `CronHandler` type (from types.go)
- [ ] `CronPayload` struct (from types.go)

**Functions:**
- [ ] `calculateNextRun(...)` (from types.go)

### Files to keep as-is:
- [x] `models.go` - Type aliases only
- [x] `store_test.go` - Store unit tests
- [x] `test_helpers.go` - Test utilities (rename functions)
- [x] `hydra_test.go` - Integration tests (rename to engine_test.go)
- [x] `testharness/` - Test utilities

## Migration Steps

1. [x] Create `engine.go` and move Engine-related code
2. [x] Move WorkerConfig to `worker.go`
3. [x] Create `workflow.go` and move workflow-related code
4. [x] Create `cron.go` and move cron-related code
5. [x] Create `sleep.go` and move Sleep function
6. [x] Clean up `types.go` to contain only core types
7. [x] Remove unused types (`WorkflowHandler`) and replace `Payload` with `any`
8. [x] Remove empty `types.go` file entirely
9. [x] Move `Worker` interface to `worker.go`
10. [x] Update imports in all files
11. [x] Rename `hydra_test.go` to `engine_test.go`
12. [x] Test that everything compiles and tests pass
13. [x] Clean up any remaining duplicate definitions (removed hydra.go)

## Success Criteria
- [x] All tests pass
- [x] Clear separation of concerns
- [x] Each file has a single responsibility
- [x] Easy to find where specific functionality lives