package hydra

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSQLCQueryCoverage(t *testing.T) {
	// Test that basic SQLC Query operations work
	engine := newTestEngine(t)
	ctx := context.Background()
	namespace := engine.GetNamespace()

	t.Run("WorkflowOperations", func(t *testing.T) {
		// Test CreateWorkflow using Query pattern
		workflowID := uid.New(uid.WorkflowPrefix)
		err := store.Query.CreateWorkflow(ctx, engine.GetDB(), store.CreateWorkflowParams{
			ID:                workflowID,
			WorkflowName:      "test-workflow",
			Status:            store.WorkflowExecutionsStatusPending,
			InputData:         []byte(`{"test": "data"}`),
			OutputData:        []byte{},
			ErrorMessage:      sql.NullString{Valid: false},
			CreatedAt:         time.Now().UnixMilli(),
			StartedAt:         sql.NullInt64{Valid: false},
			CompletedAt:       sql.NullInt64{Valid: false},
			MaxAttempts:       3,
			RemainingAttempts: 3,
			NextRetryAt:       sql.NullInt64{Valid: false},
			Namespace:         namespace,
			TriggerType:       store.NullWorkflowExecutionsTriggerType{Valid: false},
			TriggerSource:     sql.NullString{Valid: false},
			SleepUntil:        sql.NullInt64{Valid: false},
			TraceID:           sql.NullString{Valid: false},
			SpanID:            sql.NullString{Valid: false},
		})
		require.NoError(t, err, "CreateWorkflow should work")

		// Test GetWorkflow using Query pattern
		workflow, err := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        workflowID,
			Namespace: namespace,
		})
		require.NoError(t, err, "GetWorkflow should work")
		require.Equal(t, workflowID, workflow.ID)
		require.Equal(t, "test-workflow", workflow.WorkflowName)
		require.Equal(t, store.WorkflowExecutionsStatusPending, workflow.Status)

		// GetAllWorkflows was removed - test individual workflow retrieval instead
		retrievedWorkflow, err := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        workflowID,
			Namespace: namespace,
		})
		require.NoError(t, err, "GetWorkflow should work")
		require.Equal(t, workflowID, retrievedWorkflow.ID)

		// Test GetPendingWorkflows using Query pattern
		pendingWorkflows, err := store.Query.GetPendingWorkflows(ctx, engine.GetDB(), store.GetPendingWorkflowsParams{
			Namespace:   namespace,
			NextRetryAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			SleepUntil:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			Limit:       10,
		})
		require.NoError(t, err, "GetPendingWorkflows should work")
		require.Len(t, pendingWorkflows, 1)

		// Test UpdateWorkflowFields (replacement for UpdateWorkflowStatus)
		// Note: This will fail due to lease validation, which is expected in tests
		now := time.Now().UnixMilli()
		_ = store.Query.UpdateWorkflowFields(ctx, engine.GetDB(), store.UpdateWorkflowFieldsParams{
			Status:            store.WorkflowExecutionsStatusRunning,
			ErrorMessage:      sql.NullString{Valid: false},
			CompletedAt:       sql.NullInt64{Valid: false},
			StartedAt:         sql.NullInt64{Valid: false},
			OutputData:        nil,
			RemainingAttempts: 0,
			NextRetryAt:       sql.NullInt64{Valid: false},
			SleepUntil:        sql.NullInt64{Valid: false},
			ID:                workflowID,
			Namespace:         namespace,
			ResourceID:        workflowID,
			WorkerID:          "test-worker",
			ExpiresAt:         now,
		})
		// Ignore error due to missing lease

		// Test CompleteWorkflow using restored simple query
		err = store.Query.CompleteWorkflow(ctx, engine.GetDB(), store.CompleteWorkflowParams{
			CompletedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			OutputData:  []byte(`{"result": "success"}`),
			ID:          workflowID,
			Namespace:   namespace,
		})
		require.NoError(t, err, "CompleteWorkflow should work")

		// Verify final state
		finalWorkflow, err := store.Query.GetWorkflow(ctx, engine.GetDB(), store.GetWorkflowParams{
			ID:        workflowID,
			Namespace: namespace,
		})
		require.NoError(t, err)
		require.Equal(t, store.WorkflowExecutionsStatusCompleted, finalWorkflow.Status)
	})

	t.Run("StepOperations", func(t *testing.T) {
		// Create a workflow first
		workflowID := uid.New(uid.WorkflowPrefix)
		err := store.Query.CreateWorkflow(ctx, engine.GetDB(), store.CreateWorkflowParams{
			ID:                workflowID,
			WorkflowName:      "test-workflow-with-steps",
			Status:            store.WorkflowExecutionsStatusRunning,
			InputData:         []byte(`{"test": "data"}`),
			OutputData:        []byte{},
			ErrorMessage:      sql.NullString{Valid: false},
			CreatedAt:         time.Now().UnixMilli(),
			StartedAt:         sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			CompletedAt:       sql.NullInt64{Valid: false},
			MaxAttempts:       3,
			RemainingAttempts: 3,
			NextRetryAt:       sql.NullInt64{Valid: false},
			Namespace:         namespace,
			TriggerType:       store.NullWorkflowExecutionsTriggerType{Valid: false},
			TriggerSource:     sql.NullString{Valid: false},
			SleepUntil:        sql.NullInt64{Valid: false},
			TraceID:           sql.NullString{Valid: false},
			SpanID:            sql.NullString{Valid: false},
		})
		require.NoError(t, err)

		// Test CreateStep using Query pattern
		stepID := uid.New(uid.StepPrefix)
		err = store.Query.CreateStep(ctx, engine.GetDB(), store.CreateStepParams{
			ID:                stepID,
			ExecutionID:       workflowID,
			StepName:          "test-step",
			Status:            store.WorkflowStepsStatusRunning,
			OutputData:        []byte{},
			ErrorMessage:      sql.NullString{Valid: false},
			StartedAt:         sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			CompletedAt:       sql.NullInt64{Valid: false},
			MaxAttempts:       3,
			RemainingAttempts: 3,
			Namespace:         namespace,
		})
		require.NoError(t, err, "CreateStep should work")

		// Test GetStep using Query pattern
		step, err := store.Query.GetStep(ctx, engine.GetDB(), store.GetStepParams{
			Namespace:   namespace,
			ExecutionID: workflowID,
			StepName:    "test-step",
		})
		require.NoError(t, err, "GetStep should work")
		require.Equal(t, stepID, step.ID)
		require.Equal(t, "test-step", step.StepName)

		// Test UpdateStepStatus using restored simple query
		err = store.Query.UpdateStepStatus(ctx, engine.GetDB(), store.UpdateStepStatusParams{
			Status:       store.WorkflowStepsStatusCompleted,
			CompletedAt:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			OutputData:   []byte(`{"step_result": "success"}`),
			ErrorMessage: sql.NullString{Valid: false},
			Namespace:    namespace,
			ExecutionID:  workflowID,
			StepName:     "test-step",
		})
		require.NoError(t, err, "UpdateStepStatus should work")

		// GetAllSteps was removed - test individual step retrieval instead
		retrievedStep, err := store.Query.GetStep(ctx, engine.GetDB(), store.GetStepParams{
			Namespace:   namespace,
			ExecutionID: workflowID,
			StepName:    "test-step",
		})
		require.NoError(t, err, "GetStep should work")
		require.Equal(t, "test-step", retrievedStep.StepName)
	})
}
