package hydra

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// TestStepExecutionAtomicity ensures that step execution is atomic:
// either a step fully completes (executes + status update) or it doesn't execute at all.
// This prevents duplicate side effects when status updates fail after step execution.
func TestStepExecutionAtomicity(t *testing.T) {
	// Arrange: Create engine with test clock and a workflow that tracks execution attempts
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	var stepExecutionCount int64
	var sideEffectsCount int64 // Track side effects that should only happen once

	// Create a workflow with a step that has side effects
	workflow := &atomicityTestWorkflow{
		engine: engine,
		name:   "atomicity-test-workflow",
		stepFunc: func(ctx context.Context) (string, error) {
			// This represents the step execution with side effects
			_ = atomic.AddInt64(&stepExecutionCount, 1)

			// Simulate important side effects (e.g., sending email, charging payment, etc.)
			atomic.AddInt64(&sideEffectsCount, 1)

			return "step-result", nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Act: Start workflow
	executionID, err := workflow.Start(ctx, struct{}{})
	require.NoError(t, err)

	// Start worker
	worker, err := NewWorker(engine, WorkerConfig{
		Concurrency:       1,
		PollInterval:      100 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second,
		ClaimTimeout:      5 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Trigger workflow execution
	require.Eventually(t, func() bool {
		testClock.Tick(200 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)

		// Check if workflow completed
		currentStatus, getErr := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        executionID,
			Namespace: engine.GetNamespace(),
		})
		if getErr != nil {
			return false
		}
		return currentStatus.Status == store.WorkflowExecutionsStatusCompleted
	}, 5*time.Second, 50*time.Millisecond, "Workflow should complete")

	// Assert: Step should execute exactly once despite any potential failures
	finalExecutionCount := atomic.LoadInt64(&stepExecutionCount)
	finalSideEffectsCount := atomic.LoadInt64(&sideEffectsCount)

	require.Equal(t, int64(1), finalExecutionCount,
		"ATOMICITY VIOLATION: Step executed %d times instead of 1. "+
			"This indicates non-atomic step execution where the step ran multiple times.", finalExecutionCount)

	require.Equal(t, int64(1), finalSideEffectsCount,
		"SIDE EFFECT DUPLICATION: Side effects occurred %d times instead of 1. "+
			"This could mean duplicate emails sent, multiple payments charged, etc.", finalSideEffectsCount)

	// Verify the workflow completed successfully
	finalWorkflow, err := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
		ID:        executionID,
		Namespace: engine.GetNamespace(),
	})
	require.NoError(t, err)
	require.Equal(t, store.WorkflowExecutionsStatusCompleted, finalWorkflow.Status,
		"Workflow should complete successfully")

}

// TestStepExecutionAtomicityWithFailures tests atomicity when database operations fail
func TestStepExecutionAtomicityWithFailures(t *testing.T) {
	// This test would be more complex and would require mocking the store
	// to simulate failures during status updates after step execution.
	// For now, we'll focus on the basic atomicity test above.
	t.Skip("TODO: Implement test with simulated database failures during status updates")
}

// TestConcurrentStepExecution tests that multiple workers don't execute the same step
func TestConcurrentStepExecution(t *testing.T) {
	// Arrange: Create engine with test clock
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	var stepExecutionCount int64

	// Create a workflow with a step that takes time to execute
	workflow := &atomicityTestWorkflow{
		engine: engine,
		name:   "concurrent-test-workflow",
		stepFunc: func(ctx context.Context) (string, error) {
			_ = atomic.AddInt64(&stepExecutionCount, 1)

			// Simulate some work time
			time.Sleep(100 * time.Millisecond)

			return "concurrent-result", nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start workflow
	executionID, err := workflow.Start(ctx, struct{}{})
	require.NoError(t, err)

	// Start multiple workers that might try to process the same workflow
	worker1ID := uid.New(uid.WorkerPrefix)
	worker2ID := uid.New(uid.WorkerPrefix)

	worker1, err := NewWorker(engine, WorkerConfig{
		WorkerID:          worker1ID,
		Concurrency:       1,
		PollInterval:      50 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second,
		ClaimTimeout:      5 * time.Second,
	})
	require.NoError(t, err)

	worker2, err := NewWorker(engine, WorkerConfig{
		WorkerID:          worker2ID,
		Concurrency:       1,
		PollInterval:      50 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second,
		ClaimTimeout:      5 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker1, workflow)
	require.NoError(t, err)
	err = RegisterWorkflow(worker2, workflow)
	require.NoError(t, err)

	err = worker1.Start(ctx)
	require.NoError(t, err)
	defer worker1.Shutdown(ctx)

	err = worker2.Start(ctx)
	require.NoError(t, err)
	defer worker2.Shutdown(ctx)

	// Trigger both workers to poll simultaneously
	require.Eventually(t, func() bool {
		testClock.Tick(100 * time.Millisecond)
		time.Sleep(20 * time.Millisecond)

		// Check if workflow completed
		currentStatus, err := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        executionID,
			Namespace: engine.GetNamespace(),
		})
		if err != nil {
			return false
		}
		return currentStatus.Status == store.WorkflowExecutionsStatusCompleted
	}, 5*time.Second, 50*time.Millisecond, "Workflow should complete with concurrent workers")

	// Assert: Step should execute exactly once even with multiple workers
	finalExecutionCount := atomic.LoadInt64(&stepExecutionCount)
	require.Equal(t, int64(1), finalExecutionCount,
		"CONCURRENCY VIOLATION: Step executed %d times instead of 1. "+
			"Multiple workers executed the same step, violating exactly-once guarantees.", finalExecutionCount)

}

// atomicityTestWorkflow is a test workflow for testing step execution atomicity
type atomicityTestWorkflow struct {
	engine   *Engine
	name     string
	stepFunc func(ctx context.Context) (string, error)
}

func (w *atomicityTestWorkflow) Name() string {
	return w.name
}

func (w *atomicityTestWorkflow) Run(ctx WorkflowContext, req any) error {
	_, err := Step(ctx, "atomic-step", w.stepFunc)
	return err
}

func (w *atomicityTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
