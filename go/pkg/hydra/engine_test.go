package hydra

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/testharness"
)

// Test workflow that counts executions and emits events
type CountingWorkflow struct {
	counter *int64
	events  *testharness.EventCollector
}

func (w *CountingWorkflow) Name() string {
	return "counting-workflow"
}

func (w *CountingWorkflow) Run(ctx WorkflowContext, req struct{}) error {
	w.events.Emit(ctx, testharness.WorkflowStarted, "Starting counting workflow")

	_, err := Step(ctx, "increment", func(stepCtx context.Context) (string, error) {
		w.events.Emit(ctx, testharness.StepExecuting, "Executing increment step", "step_name", "increment")

		// This should only execute exactly once
		atomic.AddInt64(w.counter, 1)

		w.events.Emit(ctx, testharness.StepExecuted, "Completed increment step", "step_name", "increment", "result", "incremented")

		return "incremented", nil
	})

	if err != nil {
		w.events.Emit(ctx, testharness.WorkflowFailed, "Workflow failed", "error", err.Error())
	} else {
		w.events.Emit(ctx, testharness.WorkflowCompleted, "Workflow completed successfully")
	}

	return err
}

// CRITICAL CORRECTNESS TESTS

func TestBasicWorkflowRegistration(t *testing.T) {
	// Given: An engine instance and workflow
	e := newTestEngine(t)
	events := testharness.NewEventCollector()
	workflow := &CountingWorkflow{
		counter: new(int64),
		events:  events,
	}

	// When: Creating worker and registering workflow
	worker, err := NewWorker(e, WorkerConfig{
		Concurrency: 1,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(context.Background())

	// Then: Worker should start without error
	require.NoError(t, err)
	require.NotNil(t, worker)
	defer worker.Shutdown(context.Background())

	// And: We should be able to start a workflow
	executionID, err := e.StartWorkflow(context.Background(), workflow.Name(), struct{}{})
	require.NoError(t, err)
	require.NotEmpty(t, executionID)
}

func TestStepExecutesExactlyOnce(t *testing.T) {
	// Given: A workflow with a step that increments a counter and emits events
	testClock := clock.NewTestClock()
	e := newTestEngineWithClock(t, testClock)
	events := testharness.NewEventCollector()
	counter := int64(0)
	workflow := &CountingWorkflow{
		counter: &counter,
		events:  events,
	}

	// When: Creating worker, registering workflow, and starting
	worker, err := NewWorker(e, WorkerConfig{
		Concurrency:  1,
		PollInterval: 100 * time.Millisecond, // Fast polling for test
	})
	require.NoError(t, err)
	defer worker.Shutdown(context.Background())

	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	// Give worker time to start polling
	time.Sleep(50 * time.Millisecond)

	// Start workflow execution
	executionID, err := e.StartWorkflow(context.Background(), workflow.Name(), struct{}{})
	require.NoError(t, err)
	require.NotEmpty(t, executionID)

	// Trigger worker polling with test clock
	for i := 0; i < 10; i++ {
		testClock.Tick(200 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)

		// Check if workflow has been picked up
		currentStatus, err := store.Query.GetWorkflow(context.Background(), e.GetDB(), store.GetWorkflowParams{
			ID:        executionID,
			Namespace: e.GetNamespace(),
		})
		require.NoError(t, err)
		if currentStatus.Status != store.WorkflowExecutionsStatusPending {
			break
		}
	}

	// Wait for completion
	completedWorkflow := waitForWorkflowCompletion(t, e, executionID, 3*time.Second)
	require.NotNil(t, completedWorkflow)

	// Then: Assert using both counter and events
	finalCount := atomic.LoadInt64(&counter)

	// Check events for detailed analysis
	stepExecutions := events.FilterWithData(testharness.StepExecuting, "step_name", "increment")
	stepCompletions := events.FilterWithData(testharness.StepExecuted, "step_name", "increment")
	workflowCompletions := events.Filter(testharness.WorkflowCompleted)

	// The critical assertion: step should execute exactly once
	assert.Equal(t, int64(1), finalCount, "Counter should be incremented exactly once")
	assert.Equal(t, 1, len(stepExecutions), "Step should be executed exactly once")
	assert.Equal(t, 1, len(stepCompletions), "Step should complete exactly once")
	assert.Equal(t, 1, len(workflowCompletions), "Workflow should complete exactly once")
}

func TestStepCheckpointingPreventsReExecution(t *testing.T) {
	t.Skip("TODO: Implement checkpointing test")
}

func TestWorkflowTerminatesEventually(t *testing.T) {
	t.Skip("TODO: Implement retry limit testing")
}

func TestWorkerCrashRecovery(t *testing.T) {
	t.Skip("TODO: Implement worker crash recovery testing")
}

func TestNoDuplicateStepExecution(t *testing.T) {
	t.Skip("TODO: Implement concurrency safety testing")
}
