package hydra

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/testharness"
)

// TestSimpleDataConsistency tests basic data consistency using event collection
func TestSimpleDataConsistency(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	const numWorkflows = 10

	// Create event collector
	eventCollector := testharness.NewEventCollector()

	// Create event-aware workflow
	workflow := &eventTrackingWorkflow{
		engine:    engine,
		name:      "simple-consistency-workflow",
		collector: eventCollector,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	// Start a single worker
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "simple-consistency-worker",
		Concurrency:       2,
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

	// Submit workflows
	workflowIDs := make([]string, numWorkflows)
	for i := 0; i < numWorkflows; i++ {
		workflowID, startErr := workflow.Start(ctx, fmt.Sprintf("payload-%d", i))
		require.NoError(t, startErr)
		workflowIDs[i] = workflowID
	}

	// Wait for all workflows to finish using event collection
	require.Eventually(t, func() bool {
		completedEvents := eventCollector.Count(testharness.WorkflowCompleted)
		failedEvents := eventCollector.Count(testharness.WorkflowFailed)
		totalFinished := completedEvents + failedEvents

		return totalFinished == numWorkflows
	}, 10*time.Second, 200*time.Millisecond, "All workflows should finish")

	// Verify exactly-once execution using events
	for _, workflowID := range workflowIDs {
		// Verify exactly one started event
		startedEvents := eventCollector.FilterWithData(testharness.WorkflowStarted, "execution_id", workflowID)
		require.Len(t, startedEvents, 1, "Workflow %s should start exactly once", workflowID)

		// Verify exactly one completion event
		completedEvents := eventCollector.FilterWithData(testharness.WorkflowCompleted, "execution_id", workflowID)
		failedEvents := eventCollector.FilterWithData(testharness.WorkflowFailed, "execution_id", workflowID)
		totalCompletions := len(completedEvents) + len(failedEvents)
		require.Equal(t, 1, totalCompletions, "Workflow %s should complete exactly once", workflowID)

		// Verify exactly one step execution
		stepExecutingEvents := eventCollector.FilterWithData(testharness.StepExecuting, "execution_id", workflowID)
		require.Len(t, stepExecutingEvents, 1, "Workflow %s should have exactly one step execution", workflowID)
	}

	// Verify database consistency
	// GetAllWorkflows was removed - check completed workflows indirectly
	// Since we know we created specific workflows, verify them individually
	completedInDB := 0
	for _, id := range workflowIDs {
		wf, err := store.Query.GetWorkflow(context.Background(), engine.GetDB(), store.GetWorkflowParams{
			ID:        id,
			Namespace: engine.GetNamespace(),
		})
		if err == nil && wf.Status == store.WorkflowExecutionsStatusCompleted {
			completedInDB++
		}
	}

	completedEventsCount := eventCollector.Count(testharness.WorkflowCompleted)
	require.Equal(t, completedEventsCount, completedInDB,
		"Database completed count should match completed events")

}

// TestConcurrentWorkerConsistency tests consistency with multiple workers using events
func TestConcurrentWorkerConsistency(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	const (
		numWorkers   = 3
		numWorkflows = 15
	)

	// Create event collector
	eventCollector := testharness.NewEventCollector()

	// Create event-aware workflow
	workflow := &eventTrackingWorkflow{
		engine:    engine,
		name:      "concurrent-consistency-workflow",
		collector: eventCollector,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Start multiple workers
	workers := make([]Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker, err := NewWorker(engine, WorkerConfig{
			WorkerID:          fmt.Sprintf("concurrent-worker-%d", i),
			Concurrency:       2,
			PollInterval:      50 * time.Millisecond,
			HeartbeatInterval: 1 * time.Second,
			ClaimTimeout:      5 * time.Second,
		})
		require.NoError(t, err)

		err = RegisterWorkflow(worker, workflow)
		require.NoError(t, err)

		err = worker.Start(ctx)
		require.NoError(t, err)
		defer worker.Shutdown(ctx)

		workers[i] = worker
	}

	// Submit workflows
	workflowIDs := make([]string, numWorkflows)
	for i := 0; i < numWorkflows; i++ {
		workflowID, err := workflow.Start(ctx, fmt.Sprintf("concurrent-payload-%d", i))
		require.NoError(t, err)
		workflowIDs[i] = workflowID
	}

	// Wait for workflows to finish using event collection
	require.Eventually(t, func() bool {
		completedEvents := eventCollector.Count(testharness.WorkflowCompleted)
		failedEvents := eventCollector.Count(testharness.WorkflowFailed)
		totalFinished := completedEvents + failedEvents

		return totalFinished == numWorkflows
	}, 12*time.Second, 300*time.Millisecond, "All concurrent workflows should finish")

	// Verify exactly-once execution for each workflow
	duplicateExecutions := 0
	duplicateCompletions := 0

	for _, workflowID := range workflowIDs {
		// Check for duplicate workflow executions
		startedEvents := eventCollector.FilterWithData(testharness.WorkflowStarted, "execution_id", workflowID)
		if len(startedEvents) > 1 {
			duplicateExecutions++
			t.Errorf("DUPLICATE EXECUTION: Workflow %s started %d times", workflowID, len(startedEvents))
		}
		require.Len(t, startedEvents, 1, "Workflow %s should start exactly once", workflowID)

		// Check for duplicate completions
		completedEvents := eventCollector.FilterWithData(testharness.WorkflowCompleted, "execution_id", workflowID)
		failedEvents := eventCollector.FilterWithData(testharness.WorkflowFailed, "execution_id", workflowID)
		totalCompletions := len(completedEvents) + len(failedEvents)

		if totalCompletions > 1 {
			duplicateCompletions++
			t.Errorf("DUPLICATE COMPLETION: Workflow %s completed %d times (%d completed + %d failed)",
				workflowID, totalCompletions, len(completedEvents), len(failedEvents))
		}
		require.Equal(t, 1, totalCompletions, "Workflow %s should complete exactly once", workflowID)

		// Verify exactly one step execution
		stepExecutingEvents := eventCollector.FilterWithData(testharness.StepExecuting, "execution_id", workflowID)
		require.Len(t, stepExecutingEvents, 1, "Workflow %s should have exactly one step execution", workflowID)
	}

	// Assert no duplicates were found
	require.Equal(t, 0, duplicateExecutions, "Should have zero duplicate workflow executions")
	require.Equal(t, 0, duplicateCompletions, "Should have zero duplicate workflow completions")

	// Verify database consistency
	// GetAllWorkflows was removed - check completed workflows indirectly
	// Since we know we created specific workflows, verify them individually
	completedInDB := 0
	for _, id := range workflowIDs {
		wf, err := store.Query.GetWorkflow(context.Background(), engine.GetDB(), store.GetWorkflowParams{
			ID:        id,
			Namespace: engine.GetNamespace(),
		})
		if err == nil && wf.Status == store.WorkflowExecutionsStatusCompleted {
			completedInDB++
		}
	}

	completedEventsCount := eventCollector.Count(testharness.WorkflowCompleted)
	require.Equal(t, completedEventsCount, completedInDB,
		"Database completed count should match completed events")

}

// eventTrackingWorkflow emits events during execution for testing
type eventTrackingWorkflow struct {
	engine    *Engine
	name      string
	collector *testharness.EventCollector
}

func (w *eventTrackingWorkflow) Name() string {
	return w.name
}

func (w *eventTrackingWorkflow) Run(ctx WorkflowContext, req any) error {
	// Emit workflow started event
	w.collector.Emit(ctx, testharness.WorkflowStarted, "Workflow execution started")

	// Emit step executing event
	w.collector.Emit(ctx, testharness.StepExecuting, "Step execution started", "step_name", "consistency-step")

	// Execute the step
	result, err := Step(ctx, "consistency-step", func(stepCtx context.Context) (string, error) {
		// Simulate some work
		time.Sleep(20 * time.Millisecond)
		return "step-completed", nil
	})

	if err != nil {
		// Emit step failed event
		w.collector.Emit(ctx, testharness.StepFailed, "Step execution failed",
			"step_name", "consistency-step", "error", err.Error())

		// Emit workflow failed event
		w.collector.Emit(ctx, testharness.WorkflowFailed, "Workflow execution failed", "error", err.Error())

		return err
	}

	// Emit step executed event
	w.collector.Emit(ctx, testharness.StepExecuted, "Step execution completed",
		"step_name", "consistency-step", "result", result)

	// Emit workflow completed event
	w.collector.Emit(ctx, testharness.WorkflowCompleted, "Workflow execution completed")

	return nil
}

func (w *eventTrackingWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
