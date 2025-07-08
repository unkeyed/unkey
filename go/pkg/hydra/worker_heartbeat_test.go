package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// TestWorkerHeartbeatFunctionality ensures that workers send heartbeats to maintain their leases
// and prevent workflows from being incorrectly marked as orphaned when workers are healthy.
func TestWorkerHeartbeatFunctionality(t *testing.T) {
	// Arrange: Create engine with test clock for deterministic timing
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	// Create a workflow that will run for a while to give us time to test heartbeats
	workflow := &longRunningWorkflow{
		engine:      engine,
		name:        "heartbeat-test-workflow",
		executeTime: 5 * time.Second, // Run longer than heartbeat interval
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start workflow
	executionID, err := workflow.Start(ctx, struct{}{})
	require.NoError(t, err)

	// Start worker with short heartbeat interval for faster testing
	workerID := uid.New(uid.WorkerPrefix)
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          workerID,
		Concurrency:       1,
		PollInterval:      100 * time.Millisecond,
		HeartbeatInterval: 1 * time.Second,  // Send heartbeats frequently
		ClaimTimeout:      10 * time.Second, // Long enough for multiple heartbeats
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Act: Let worker pick up workflow and start sending heartbeats
	testClock.Tick(200 * time.Millisecond) // Trigger initial poll
	time.Sleep(50 * time.Millisecond)      // Let worker pick up the workflow

	// Keep triggering polls until workflow is picked up
	require.Eventually(t, func() bool {
		testClock.Tick(200 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)

		// Check if workflow has been picked up
		currentStatus, getErr := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        executionID,
			Namespace: engine.GetNamespace(),
		})
		if getErr != nil {
			return false
		}
		return currentStatus.Status != store.WorkflowExecutionsStatusPending
	}, 3*time.Second, 50*time.Millisecond, "Worker should pick up workflow within timeout")

	// Verify workflow is being processed
	workflowStatus, err := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
		ID:        executionID,
		Namespace: engine.GetNamespace(),
	})
	require.NoError(t, err)
	require.Equal(t, store.WorkflowExecutionsStatusRunning, workflowStatus.Status, "Workflow should be running")

	// Get initial lease
	lease, err := store.Query.GetLease(ctx, engine.GetDB(), store.GetLeaseParams{
		ResourceID: executionID,
		Kind:       store.LeasesKindWorkflow,
	})
	require.NoError(t, err)
	require.Equal(t, workerID, lease.WorkerID, "Lease should be held by our worker")

	initialExpiresAt := lease.ExpiresAt

	// Advance time to trigger first heartbeat
	testClock.Tick(1500 * time.Millisecond) // Past first heartbeat interval
	time.Sleep(50 * time.Millisecond)       // Let heartbeat be processed

	// Verify heartbeat extended the lease
	updatedLease, err := store.Query.GetLease(ctx, engine.GetDB(), store.GetLeaseParams{
		ResourceID: executionID,
		Kind:       store.LeasesKindWorkflow,
	})
	require.NoError(t, err)
	require.Equal(t, workerID, updatedLease.WorkerID, "Lease should still be held by our worker")
	require.Greater(t, updatedLease.ExpiresAt, initialExpiresAt,
		"HEARTBEAT FAILURE: Lease expiration should be extended after heartbeat. "+
			"Initial: %d, Updated: %d. This means the worker is not sending heartbeats properly, "+
			"which could cause healthy workers to lose their leases prematurely.",
		initialExpiresAt, updatedLease.ExpiresAt)
	require.Greater(t, updatedLease.HeartbeatAt, lease.HeartbeatAt,
		"HeartbeatAt timestamp should be updated")

	// The key test: verify heartbeat actually extended the lease
	extensionAmount := updatedLease.ExpiresAt - initialExpiresAt
	require.Greater(t, extensionAmount, int64(0),
		"HEARTBEAT SUCCESS: Lease was extended by %d ms. Heartbeats are working correctly.", extensionAmount)

}

// longRunningWorkflow simulates a workflow that takes time to execute,
// giving us opportunity to test heartbeat behavior during execution
type longRunningWorkflow struct {
	engine      *Engine
	name        string
	executeTime time.Duration
}

func (w *longRunningWorkflow) Name() string {
	return w.name
}

func (w *longRunningWorkflow) Run(ctx WorkflowContext, req any) error {
	// Simulate long-running work by sleeping
	// In a real test, this would be actual work that takes time
	time.Sleep(w.executeTime)
	return nil
}

func (w *longRunningWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
