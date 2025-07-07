package hydra

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
)

// TestStepIdempotencyDuringWorkerFailure guarantees that workflow steps are idempotent
// and execute exactly once, even when workers fail and other workers resume the workflow.
//
// This prevents duplicate side effects like sending emails twice, processing payments
// multiple times, or creating duplicate database records during worker handoffs.
func TestStepIdempotencyDuringWorkerFailure(t *testing.T) {
	// Arrange: Create engine with test clock for deterministic timing
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	var stepExecutionCount int64

	// Create a workflow with one step that takes time to execute
	workflow := &testWorkflow{
		engine: engine,
		name:   "idempotency-test-workflow",
		stepFunc: func(ctx context.Context) (string, error) {
			// This should only execute once, but the bug causes it to execute multiple times
			atomic.AddInt64(&stepExecutionCount, 1)

			// Step executes instantly - we'll control timing via test clock and worker coordination
			return "step-completed", nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Act: Start workflow using the preferred pattern
	executionID, err := workflow.Start(ctx, struct{}{})
	require.NoError(t, err)

	// Start first worker to begin processing
	worker1, err := NewWorker(engine, WorkerConfig{
		Concurrency:       1,
		PollInterval:      50 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second, // Short heartbeat for faster cleanup
		ClaimTimeout:      2 * time.Second, // Short timeout to simulate crash
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker1, workflow)
	require.NoError(t, err)

	err = worker1.Start(ctx)
	require.NoError(t, err)

	// Let worker1 start processing - advance test clock to trigger polling
	testClock.Tick(100 * time.Millisecond) // Trigger initial poll

	// Give a brief moment for worker1 to process (this is unavoidable for goroutine coordination)
	time.Sleep(50 * time.Millisecond)

	// Keep triggering polls until workflow is picked up
	for i := 0; i < 10; i++ {
		testClock.Tick(100 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)

		// Check if workflow has been picked up
		currentStatus, getErr := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        executionID,
			Namespace: engine.GetNamespace(),
		})
		require.NoError(t, getErr)
		if currentStatus.Status != store.WorkflowExecutionsStatusPending {
			break
		}
	}

	// Check that workflow is being processed
	_, err = store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
		ID:        executionID,
		Namespace: engine.GetNamespace(),
	})
	require.NoError(t, err)

	// Simulate worker1 crash by shutting it down
	err = worker1.Shutdown(context.Background())
	require.NoError(t, err)

	// Advance time to expire the lease and trigger cleanup
	// Leases expire after ClaimTimeout (2 seconds), cleanup runs every HeartbeatInterval * 2 (2 seconds)
	testClock.Tick(3 * time.Second) // Advance past lease expiration + cleanup interval

	// Start worker2 to take over the workflow
	worker2, err := NewWorker(engine, WorkerConfig{
		Concurrency:       1,
		PollInterval:      50 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second, // Short heartbeat for faster cleanup
		ClaimTimeout:      5 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker2, workflow)
	require.NoError(t, err)

	err = worker2.Start(ctx)
	require.NoError(t, err)
	defer worker2.Shutdown(ctx)

	// Advance time to trigger worker2 polling and cleanup detection
	testClock.Tick(200 * time.Millisecond) // Trigger worker2 polling

	// Keep triggering polls and cleanup until workflow is picked up by worker2
	for i := 0; i < 20; i++ {
		testClock.Tick(200 * time.Millisecond) // Trigger polling and cleanup
		time.Sleep(10 * time.Millisecond)

		// Check if workflow has been picked up
		currentStatus, err := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        executionID,
			Namespace: engine.GetNamespace(),
		})
		require.NoError(t, err)
		if currentStatus.Status != store.WorkflowExecutionsStatusPending {
			break
		}
	}

	// Wait for worker2 to complete the workflow
	finalResult := waitForWorkflowCompletion(t, engine, executionID, 3*time.Second)
	// Check final step execution count
	finalCount := atomic.LoadInt64(&stepExecutionCount)

	// Assert: Step idempotency - should execute exactly once despite worker failure
	require.Equal(t, int64(1), finalCount,
		"STEP IDEMPOTENCY VIOLATION: Step executed %d times instead of 1. "+
			"This could cause duplicate side effects like sending emails twice, "+
			"processing payments multiple times, or creating duplicate records.", finalCount)

	// Verify workflow completed successfully
	require.Equal(t, store.WorkflowExecutionsStatusCompleted, finalResult.Status, "Workflow should complete successfully despite worker crash")
}

// testWorkflow is a minimal workflow for testing step idempotency
type testWorkflow struct {
	engine   *Engine
	name     string
	stepFunc func(ctx context.Context) (string, error)
}

func (w *testWorkflow) Name() string {
	return w.name
}

func (w *testWorkflow) Run(ctx WorkflowContext, req any) error {
	_, err := Step(ctx, "test-step", w.stepFunc)
	return err
}

// Start is a convenience method that starts this workflow using the embedded engine
// This encourages a cleaner API pattern: workflow.Start() instead of engine.StartWorkflow()
func (w *testWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
