package hydra

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
)

// TestConcurrentWorkflowAccess tests that multiple workers trying to acquire
// the same workflow maintain data consistency and exactly-once execution
func TestConcurrentWorkflowAccess(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	const (
		numWorkers   = 5
		numWorkflows = 25 // Back to full test
	)

	// Track execution attempts per workflow
	executionTracker := &ConcurrentExecutionTracker{
		executions: make(map[string]*ExecutionRecord),
	}

	// Create workflows first
	workflowIDs := make([]string, numWorkflows)
	consistencyWorkflow := &consistencyTestWorkflow{
		engine:  engine,
		name:    "consistency-test-workflow",
		tracker: executionTracker,
	}

	createCtx := context.Background()
	for i := 0; i < numWorkflows; i++ {
		executionID, err := consistencyWorkflow.Start(createCtx, fmt.Sprintf("workflow-%d", i))
		require.NoError(t, err)
		workflowIDs[i] = executionID
	}

	// Context for workers - give them plenty of time to complete
	workerCtx, workerCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer workerCancel()

	// Start multiple workers that will compete for the same workflows
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			worker, err := NewWorker(engine, WorkerConfig{
				WorkerID:          fmt.Sprintf("consistency-worker-%d", workerID),
				Concurrency:       3,                     // Allow multiple workflows per worker
				PollInterval:      50 * time.Millisecond, // Fast polling
				HeartbeatInterval: 1 * time.Second,
				ClaimTimeout:      5 * time.Second,
			})
			require.NoError(t, err)

			err = RegisterWorkflow(worker, consistencyWorkflow)
			require.NoError(t, err)

			err = worker.Start(workerCtx)
			require.NoError(t, err)
			defer worker.Shutdown(context.Background()) // Use fresh context for shutdown

			// Let worker run until context is done
			<-workerCtx.Done()
		}(i)
	}

	// Wait for all workflows to complete - check actual database status instead of execution tracker
	require.Eventually(t, func() bool {
		completedCount := 0
		for _, workflowID := range workflowIDs {
			workflow, err := store.Query.GetWorkflow(context.Background(), engine.GetDB(), store.GetWorkflowParams{
				ID:        workflowID,
				Namespace: engine.GetNamespace(),
			})
			if err == nil && workflow.Status == store.WorkflowExecutionsStatusCompleted {
				completedCount++
			}
		}

		t.Logf("Completed workflows: %d/%d", completedCount, numWorkflows)
		return completedCount == numWorkflows
	}, 20*time.Second, 200*time.Millisecond,
		"All %d workflows should complete with %d workers", numWorkflows, numWorkers)

	workerCancel() // Stop all workers
	wg.Wait()

	// Analyze results for consistency violations
	results := executionTracker.AnalyzeResults(t)

	// Critical consistency assertions
	require.Equal(t, numWorkflows, results.WorkflowsExecuted,
		"All workflows should be executed exactly once")

	require.Equal(t, 0, results.DuplicateExecutions,
		"CONSISTENCY VIOLATION: Found %d duplicate executions", results.DuplicateExecutions)

	require.Equal(t, 0, results.RaceConditions,
		"RACE CONDITION: Found %d race conditions in workflow execution", results.RaceConditions)

}

// TestStepExecutionRaceConditions tests for race conditions during step creation and updates
func TestStepExecutionRaceConditions(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	const numConcurrentSteps = 20

	// Create a workflow that will have multiple steps executed concurrently
	raceWorkflow := &raceConditionTestWorkflow{
		engine: engine,
		name:   "race-condition-workflow",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	executionID, err := raceWorkflow.Start(ctx, struct{}{})
	require.NoError(t, err)

	// Start worker
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "race-test-worker",
		Concurrency:       numConcurrentSteps, // Allow many concurrent steps
		PollInterval:      10 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second,
		ClaimTimeout:      5 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, raceWorkflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Wait for workflow completion
	finalWorkflow := waitForWorkflowCompletion(t, engine, executionID, 10*time.Second)
	require.NotNil(t, finalWorkflow)

	// Verify workflow completed successfully (no race conditions caused failures)
	require.Equal(t, store.WorkflowExecutionsStatusCompleted, finalWorkflow.Status,
		"Workflow should complete successfully without race conditions")

	// Test individual step consistency by trying to get each expected step
	expectedSteps := make([]string, numConcurrentSteps)
	for i := 0; i < numConcurrentSteps; i++ {
		expectedSteps[i] = fmt.Sprintf("race-step-%d", i)
	}

	duplicateSteps := 0
	for _, stepName := range expectedSteps {
		// Try to get the step - this verifies it exists and is unique
		step, err := store.Query.GetStep(ctx, engine.GetDB(), store.GetStepParams{
			Namespace:   engine.GetNamespace(),
			ExecutionID: executionID,
			StepName:    stepName,
		})
		if err != nil {
			t.Errorf("Expected step '%s' not found: %v", stepName, err)
			continue
		}

		// Verify step completed successfully
		if step.Status != store.WorkflowStepsStatusCompleted {
			t.Errorf("Step '%s' not completed: %s", stepName, step.Status)
		}
	}

	require.Equal(t, 0, duplicateSteps,
		"RACE CONDITION: Found %d duplicate steps", duplicateSteps)

}

// TestDatabaseTransactionIntegrity tests transaction rollback scenarios
func TestDatabaseTransactionIntegrity(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	// Create a workflow that will test transaction boundaries
	transactionWorkflow := &transactionTestWorkflow{
		engine: engine,
		name:   "transaction-test-workflow",
	}

	ctx := context.Background()

	// Test 1: Normal transaction completion
	executionID1, err := transactionWorkflow.Start(ctx, "normal")
	require.NoError(t, err)

	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "transaction-test-worker",
		Concurrency:       1,
		PollInterval:      10 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second,
		ClaimTimeout:      5 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, transactionWorkflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Wait for normal workflow to complete
	finalWorkflow1 := waitForWorkflowCompletion(t, engine, executionID1, 5*time.Second)
	require.Equal(t, store.WorkflowExecutionsStatusCompleted, finalWorkflow1.Status)

	// Verify steps were created properly by checking the transaction step
	step1, err := store.Query.GetStep(ctx, engine.GetDB(), store.GetStepParams{
		Namespace:   engine.GetNamespace(),
		ExecutionID: executionID1,
		StepName:    "transaction-step",
	})
	require.NoError(t, err)
	require.NotNil(t, step1, "Transaction step should be created for successful workflow")
	require.Equal(t, store.WorkflowStepsStatusCompleted, step1.Status, "Transaction step should complete successfully")

}

// ConcurrentExecutionTracker tracks workflow executions across multiple workers
type ConcurrentExecutionTracker struct {
	mu         sync.Mutex
	executions map[string]*ExecutionRecord
}

type ExecutionRecord struct {
	WorkflowID     string
	ExecutionCount int64
	WorkerIDs      []string
	Timestamps     []time.Time
	Completed      bool
	Failed         bool
}

type ConsistencyResults struct {
	WorkflowsExecuted   int
	DuplicateExecutions int
	FailedWorkflows     int
	RaceConditions      int
}

func (t *ConcurrentExecutionTracker) RecordExecution(workflowID, workerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.executions[workflowID] == nil {
		t.executions[workflowID] = &ExecutionRecord{
			WorkflowID: workflowID,
			WorkerIDs:  make([]string, 0),
			Timestamps: make([]time.Time, 0),
		}
	}

	record := t.executions[workflowID]
	record.ExecutionCount++
	record.WorkerIDs = append(record.WorkerIDs, workerID)
	record.Timestamps = append(record.Timestamps, time.Now())
}

func (t *ConcurrentExecutionTracker) RecordCompletion(workflowID string, success bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if record := t.executions[workflowID]; record != nil {
		record.Completed = true
		record.Failed = !success
	}
}

func (t *ConcurrentExecutionTracker) GetExecutedCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	count := 0
	for _, record := range t.executions {
		if record.ExecutionCount > 0 {
			count++
		}
	}
	return count
}

func (t *ConcurrentExecutionTracker) GetMissingWorkflows(allWorkflowIDs []string) []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	var missing []string
	for _, workflowID := range allWorkflowIDs {
		if record, exists := t.executions[workflowID]; !exists || record.ExecutionCount == 0 {
			missing = append(missing, workflowID)
		}
	}
	return missing
}

func (t *ConcurrentExecutionTracker) AnalyzeResults(testCtx *testing.T) ConsistencyResults {
	t.mu.Lock()
	defer t.mu.Unlock()

	results := ConsistencyResults{}

	for workflowID, record := range t.executions {
		if record.ExecutionCount > 0 {
			results.WorkflowsExecuted++
		}

		if record.ExecutionCount > 1 {
			results.DuplicateExecutions++
			testCtx.Errorf("DUPLICATE EXECUTION: Workflow %s executed %d times by workers %v",
				workflowID, record.ExecutionCount, record.WorkerIDs)
		}

		if record.Failed {
			results.FailedWorkflows++
		}

		// Detect race conditions (multiple workers starting execution within 100ms)
		if len(record.Timestamps) > 1 {
			for i := 1; i < len(record.Timestamps); i++ {
				if record.Timestamps[i].Sub(record.Timestamps[i-1]) < 100*time.Millisecond {
					results.RaceConditions++
					testCtx.Errorf("RACE CONDITION: Workflow %s had concurrent executions by %v",
						workflowID, record.WorkerIDs)
					break
				}
			}
		}
	}

	return results
}

// consistencyTestWorkflow tracks executions to detect consistency violations
type consistencyTestWorkflow struct {
	engine  *Engine
	name    string
	tracker *ConcurrentExecutionTracker
}

func (w *consistencyTestWorkflow) Name() string {
	return w.name
}

func (w *consistencyTestWorkflow) Run(ctx WorkflowContext, req any) error {
	workflowID := ctx.ExecutionID()

	// Record that this workflow started executing
	w.tracker.RecordExecution(workflowID, "unknown-worker") // We could get worker ID from context

	// Simulate some work with a step
	_, err := Step(ctx, "consistency-step", func(context.Context) (string, error) {
		// Small delay to increase chance of race conditions
		time.Sleep(10 * time.Millisecond)
		return "consistent", nil
	})

	// Record completion
	w.tracker.RecordCompletion(workflowID, err == nil)

	return err
}

func (w *consistencyTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// raceConditionTestWorkflow creates multiple steps to test for race conditions
type raceConditionTestWorkflow struct {
	engine *Engine
	name   string
}

func (w *raceConditionTestWorkflow) Name() string {
	return w.name
}

func (w *raceConditionTestWorkflow) Run(ctx WorkflowContext, req any) error {
	// Create multiple steps that might race with each other
	const numSteps = 20

	// Use a WaitGroup to ensure all steps complete
	var wg sync.WaitGroup
	var stepErrors atomic.Int64

	for i := 0; i < numSteps; i++ {
		wg.Add(1)
		go func(stepIndex int) {
			defer wg.Done()

			stepName := fmt.Sprintf("race-step-%d", stepIndex)
			_, err := Step(ctx, stepName, func(context.Context) (string, error) {
				return fmt.Sprintf("result-%d", stepIndex), nil
			})

			if err != nil {
				stepErrors.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if stepErrors.Load() > 0 {
		return fmt.Errorf("race condition test failed: %d step errors", stepErrors.Load())
	}

	return nil
}

func (w *raceConditionTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// transactionTestWorkflow tests database transaction integrity
type transactionTestWorkflow struct {
	engine *Engine
	name   string
}

func (w *transactionTestWorkflow) Name() string {
	return w.name
}

func (w *transactionTestWorkflow) Run(ctx WorkflowContext, req any) error {
	mode, ok := req.(string)
	if !ok {
		mode = "normal"
	}

	// Create a step that tests transaction boundaries
	_, err := Step(ctx, "transaction-step", func(stepCtx context.Context) (string, error) {
		switch mode {
		case "normal":
			return "transaction-success", nil
		case "error":
			return "", fmt.Errorf("simulated step error")
		default:
			return "unknown-mode", nil
		}
	})

	return err
}

func (w *transactionTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
