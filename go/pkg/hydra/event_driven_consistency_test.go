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

// TestEventDrivenConsistency uses the event collection system to deterministically
// verify exactly-once execution guarantees
func TestEventDrivenConsistency(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	// Create event collector
	eventCollector := testharness.NewEventCollector()

	// Create event-aware workflow
	workflow := &eventAwareWorkflow{
		engine:    engine,
		name:      "event-driven-test-workflow",
		collector: eventCollector,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Start worker
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "event-test-worker",
		Concurrency:       2,
		PollInterval:      100 * time.Millisecond,
		HeartbeatInterval: 2 * time.Second,
		ClaimTimeout:      10 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, workflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Submit workflows
	const numWorkflows = 10
	workflowIDs := make([]string, numWorkflows)

	for i := 0; i < numWorkflows; i++ {
		workflowID, err := workflow.Start(ctx, fmt.Sprintf("payload-%d", i))
		require.NoError(t, err)
		workflowIDs[i] = workflowID
	}

	// Wait for all workflows to be processed by checking events
	// This is deterministic - we wait until we see completion events for all workflows
	require.Eventually(t, func() bool {
		completedEvents := eventCollector.Count(testharness.WorkflowCompleted)
		failedEvents := eventCollector.Count(testharness.WorkflowFailed)
		totalFinished := completedEvents + failedEvents

		return totalFinished == numWorkflows
	}, 12*time.Second, 200*time.Millisecond, "All workflows should finish (complete or fail)")

	// Analyze events for consistency violations

	// Verify exactly-once execution per workflow
	for _, workflowID := range workflowIDs {
		// Each workflow should have exactly one started event
		startedEvents := eventCollector.FilterWithData(testharness.WorkflowStarted, "execution_id", workflowID)
		require.Len(t, startedEvents, 1, "Workflow %s should start exactly once", workflowID)

		// Each workflow should have exactly one completion event (completed OR failed)
		completedEvents := eventCollector.FilterWithData(testharness.WorkflowCompleted, "execution_id", workflowID)
		failedEvents := eventCollector.FilterWithData(testharness.WorkflowFailed, "execution_id", workflowID)

		totalCompletionEvents := len(completedEvents) + len(failedEvents)
		require.Equal(t, 1, totalCompletionEvents,
			"Workflow %s should have exactly one completion event, got %d completed + %d failed",
			workflowID, len(completedEvents), len(failedEvents))

		// Each step should execute exactly once
		stepExecutingEvents := eventCollector.FilterWithData(testharness.StepExecuting, "execution_id", workflowID)
		stepExecutedEvents := eventCollector.FilterWithData(testharness.StepExecuted, "execution_id", workflowID)
		stepFailedEvents := eventCollector.FilterWithData(testharness.StepFailed, "execution_id", workflowID)

		// Should have exactly one step executing event
		require.Len(t, stepExecutingEvents, 1, "Workflow %s should have exactly one step executing event", workflowID)

		// Should have exactly one step completion event (executed OR failed)
		totalStepCompletions := len(stepExecutedEvents) + len(stepFailedEvents)
		require.Equal(t, 1, totalStepCompletions,
			"Workflow %s should have exactly one step completion event, got %d executed + %d failed",
			workflowID, len(stepExecutedEvents), len(stepFailedEvents))
	}

	// Verify database consistency matches events
	allWorkflows, err := engine.store.GetAllWorkflows(ctx, "default")
	require.NoError(t, err)

	workflowStatusCounts := make(map[store.WorkflowStatus]int)
	for _, wf := range allWorkflows {
		workflowStatusCounts[wf.Status]++
	}

	// Database should match event counts
	completedInDB := workflowStatusCounts[store.WorkflowStatusCompleted]
	failedInDB := workflowStatusCounts[store.WorkflowStatusFailed]

	eventCompletedCount := eventCollector.Count(testharness.WorkflowCompleted)
	eventFailedCount := eventCollector.Count(testharness.WorkflowFailed)

	require.Equal(t, eventCompletedCount, completedInDB,
		"Completed workflows in DB should match completed events")
	require.Equal(t, eventFailedCount, failedInDB,
		"Failed workflows in DB should match failed events")

	// Verify step consistency
	allSteps, err := engine.store.GetAllSteps(ctx, "default")
	require.NoError(t, err)

	stepStatusCounts := make(map[store.StepStatus]int)
	for _, step := range allSteps {
		stepStatusCounts[step.Status]++
	}

	// Steps in database should match step events
	completedStepsInDB := stepStatusCounts[store.StepStatusCompleted]
	failedStepsInDB := stepStatusCounts[store.StepStatusFailed]

	eventStepExecutedCount := eventCollector.Count(testharness.StepExecuted)
	eventStepFailedCount := eventCollector.Count(testharness.StepFailed)

	require.Equal(t, eventStepExecutedCount, completedStepsInDB,
		"Completed steps in DB should match step executed events")
	require.Equal(t, eventStepFailedCount, failedStepsInDB,
		"Failed steps in DB should match step failed events")

}

// TestEventDrivenConcurrentConsistency tests consistency with multiple workers using events
func TestEventDrivenConcurrentConsistency(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	// Create event collector
	eventCollector := testharness.NewEventCollector()

	// Create event-aware workflow
	workflow := &eventAwareWorkflow{
		engine:    engine,
		name:      "concurrent-event-test-workflow",
		collector: eventCollector,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Start multiple workers
	const numWorkers = 3
	workers := make([]Worker, numWorkers)

	for i := 0; i < numWorkers; i++ {
		worker, err := NewWorker(engine, WorkerConfig{
			WorkerID:          fmt.Sprintf("concurrent-event-worker-%d", i),
			Concurrency:       2,
			PollInterval:      50 * time.Millisecond,
			HeartbeatInterval: 2 * time.Second,
			ClaimTimeout:      10 * time.Second,
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
	const numWorkflows = 15
	workflowIDs := make([]string, numWorkflows)

	for i := 0; i < numWorkflows; i++ {
		workflowID, err := workflow.Start(ctx, fmt.Sprintf("concurrent-payload-%d", i))
		require.NoError(t, err)
		workflowIDs[i] = workflowID
	}

	// Wait for reasonable workflow processing (focus on consistency, not completion rate)
	// Give workers time to process what they can
	time.Sleep(8 * time.Second)

	// Check what actually happened
	completedEvents := eventCollector.Count(testharness.WorkflowCompleted)
	failedEvents := eventCollector.Count(testharness.WorkflowFailed)
	totalFinished := completedEvents + failedEvents

	// Ensure we processed at least some workflows (not zero)
	require.Greater(t, totalFinished, 0, "Should process at least some workflows")

	// Verify exactly-once execution for workflows that were processed
	duplicateExecutions := 0
	duplicateCompletions := 0
	processedWorkflows := 0

	for _, workflowID := range workflowIDs {
		// Check if this workflow was processed at all
		startedEvents := eventCollector.FilterWithData(testharness.WorkflowStarted, "execution_id", workflowID)
		completedEvents := eventCollector.FilterWithData(testharness.WorkflowCompleted, "execution_id", workflowID)
		failedEvents := eventCollector.FilterWithData(testharness.WorkflowFailed, "execution_id", workflowID)

		// Skip workflows that weren't processed
		if len(startedEvents) == 0 {
			continue
		}

		processedWorkflows++

		// Check for duplicate workflow executions
		if len(startedEvents) > 1 {
			duplicateExecutions++
			t.Errorf("DUPLICATE EXECUTION: Workflow %s started %d times", workflowID, len(startedEvents))
		}

		// Check for duplicate completions
		totalCompletions := len(completedEvents) + len(failedEvents)
		if totalCompletions > 1 {
			duplicateCompletions++
			t.Errorf("DUPLICATE COMPLETION: Workflow %s completed %d times (%d completed + %d failed)",
				workflowID, totalCompletions, len(completedEvents), len(failedEvents))
		}
	}

	// Assert no duplicates
	require.Equal(t, 0, duplicateExecutions, "Should have zero duplicate workflow executions")
	require.Equal(t, 0, duplicateCompletions, "Should have zero duplicate workflow completions")

	// Verify step-level consistency for processed workflows
	duplicateStepExecutions := 0
	for _, workflowID := range workflowIDs {
		startedEvents := eventCollector.FilterWithData(testharness.WorkflowStarted, "execution_id", workflowID)
		if len(startedEvents) == 0 {
			continue // Skip unprocessed workflows
		}

		stepExecutingEvents := eventCollector.FilterWithData(testharness.StepExecuting, "execution_id", workflowID)
		if len(stepExecutingEvents) > 1 {
			duplicateStepExecutions++
			t.Errorf("DUPLICATE STEP EXECUTION: Workflow %s had %d step executing events",
				workflowID, len(stepExecutingEvents))
		}
	}

	require.Equal(t, 0, duplicateStepExecutions, "Should have zero duplicate step executions")

}

// eventAwareWorkflow emits events during execution for testing
type eventAwareWorkflow struct {
	engine    *Engine
	name      string
	collector *testharness.EventCollector
}

func (w *eventAwareWorkflow) Name() string {
	return w.name
}

func (w *eventAwareWorkflow) Run(ctx WorkflowContext, req any) error {
	// Emit workflow started event
	w.collector.Emit(ctx, testharness.WorkflowStarted, "Workflow execution started")

	// Emit step executing event
	w.collector.Emit(ctx, testharness.StepExecuting, "Step execution started", "step_name", "test-step")

	// Execute the step
	result, err := Step(ctx, "test-step", func(stepCtx context.Context) (string, error) {
		// Simulate some work
		time.Sleep(20 * time.Millisecond)
		return "step-completed", nil
	})

	if err != nil {
		// Emit step failed event
		w.collector.Emit(ctx, testharness.StepFailed, "Step execution failed",
			"step_name", "test-step", "error", err.Error())

		// Emit workflow failed event
		w.collector.Emit(ctx, testharness.WorkflowFailed, "Workflow execution failed", "error", err.Error())

		return err
	}

	// Emit step executed event
	w.collector.Emit(ctx, testharness.StepExecuted, "Step execution completed",
		"step_name", "test-step", "result", result)

	// Emit workflow completed event
	w.collector.Emit(ctx, testharness.WorkflowCompleted, "Workflow execution completed")

	return nil
}

func (w *eventAwareWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
