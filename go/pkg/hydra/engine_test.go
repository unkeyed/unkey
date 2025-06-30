package hydra

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Logf("Started workflow with execution ID: %s", executionID)
}

func TestStepExecutesExactlyOnce(t *testing.T) {
	// Given: A workflow with a step that increments a counter and emits events
	e := newTestEngine(t)
	events := testharness.NewEventCollector()
	counter := int64(0)
	workflow := &CountingWorkflow{
		counter: &counter,
		events:  events,
	}

	// When: Creating worker, registering workflow, and starting
	worker, err := NewWorker(e, WorkerConfig{
		Concurrency: 1,
	})
	require.NoError(t, err)
	defer worker.Shutdown(context.Background())
	
	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(context.Background())
	require.NoError(t, err)

	// Give worker time to start polling
	time.Sleep(100 * time.Millisecond)

	// Start workflow execution
	executionID, err := e.StartWorkflow(context.Background(), workflow.Name(), struct{}{})
	require.NoError(t, err)
	require.NotEmpty(t, executionID)

	// Check workflow was created in store
	createdWorkflow, err := e.GetStore().GetWorkflow(context.Background(), "default", executionID)
	require.NoError(t, err)
	t.Logf("Created workflow status: %s", createdWorkflow.Status)

	// Debug: Check what GetPendingWorkflows returns
	pendingWorkflows, err := e.GetStore().GetPendingWorkflows(context.Background(), "default", 10, []string{"counting-workflow"})
	require.NoError(t, err)
	t.Logf("Pending workflows found: %d", len(pendingWorkflows))
	for i, wf := range pendingWorkflows {
		t.Logf("Pending workflow %d: ID=%s, Name=%s, Status=%s", i, wf.ID, wf.WorkflowName, wf.Status)
	}

	// Wait for completion
	completedWorkflow := waitForWorkflowSuccess(t, e, executionID, 5*time.Second)
	require.NotNil(t, completedWorkflow)

	// Then: Assert using both counter and events
	finalCount := atomic.LoadInt64(&counter)

	// Check events for detailed analysis
	stepExecutions := events.FilterWithData(testharness.StepExecuting, "step_name", "increment")
	stepCompletions := events.FilterWithData(testharness.StepExecuted, "step_name", "increment")
	workflowCompletions := events.Filter(testharness.WorkflowCompleted)

	t.Logf("Events summary: %+v", events.Summary())
	t.Logf("Final counter value: %d", finalCount)
	t.Logf("Step executions: %d", len(stepExecutions))
	t.Logf("Step completions: %d", len(stepCompletions))
	t.Logf("Workflow completions: %d", len(workflowCompletions))

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

