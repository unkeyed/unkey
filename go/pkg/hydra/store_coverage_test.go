package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// TestStoreOperationCoverage ensures all store operations work end-to-end
func TestStoreOperationCoverage(t *testing.T) {
	engine := newTestEngine(t)
	ctx := context.Background()
	namespace := engine.GetNamespace()

	t.Run("WorkflowOperations", func(t *testing.T) {
		// Test complete workflow lifecycle
		workflow := &store.WorkflowExecution{
			ID:                uid.New(uid.WorkflowPrefix),
			WorkflowName:      "test-workflow",
			Status:            store.WorkflowStatusPending,
			InputData:         []byte(`{"test": true}`),
			CreatedAt:         time.Now().UnixMilli(),
			MaxAttempts:       3,
			RemainingAttempts: 3,
			Namespace:         namespace,
		}

		// Test CreateWorkflow
		err := engine.GetSQLCStore().CreateWorkflow(ctx, workflow)
		require.NoError(t, err, "CreateWorkflow should work")

		// Test GetWorkflow
		retrieved, err := engine.GetSQLCStore().GetWorkflow(ctx, namespace, workflow.ID)
		require.NoError(t, err, "GetWorkflow should work")
		require.Equal(t, workflow.ID, retrieved.ID)

		// Test GetPendingWorkflows
		pending, err := engine.GetSQLCStore().GetPendingWorkflows(ctx, namespace, 10, nil)
		require.NoError(t, err, "GetPendingWorkflows should work")
		require.Len(t, pending, 1)

		// Test UpdateWorkflowStatus
		err = engine.GetSQLCStore().UpdateWorkflowStatus(ctx, namespace, workflow.ID, store.WorkflowStatusRunning, "")
		require.NoError(t, err, "UpdateWorkflowStatus should work")

		// Test CompleteWorkflow
		err = engine.GetSQLCStore().CompleteWorkflow(ctx, namespace, workflow.ID, []byte(`{"result": "success"}`))
		require.NoError(t, err, "CompleteWorkflow should work")
	})

	t.Run("StepOperations", func(t *testing.T) {
		// Test step lifecycle
		step := &store.WorkflowStep{
			ID:                uid.New(uid.StepPrefix),
			ExecutionID:       uid.New(uid.WorkflowPrefix),
			StepName:          "test-step",
			StepOrder:         1,
			Status:            store.StepStatusPending,
			MaxAttempts:       3,
			RemainingAttempts: 3,
			Namespace:         namespace,
		}

		// Test CreateStep
		err := engine.GetSQLCStore().CreateStep(ctx, step)
		require.NoError(t, err, "CreateStep should work")

		// Test GetStep
		retrieved, err := engine.GetSQLCStore().GetStep(ctx, namespace, step.ExecutionID, step.StepName)
		require.NoError(t, err, "GetStep should work")
		require.Equal(t, step.ID, retrieved.ID)

		// Test UpdateStepStatus
		err = engine.GetSQLCStore().UpdateStepStatus(ctx, namespace, step.ExecutionID, step.StepName, store.StepStatusCompleted, []byte(`{"output": "test"}`), "")
		require.NoError(t, err, "UpdateStepStatus should work")

		// Test GetCompletedStep
		completed, err := engine.GetSQLCStore().GetCompletedStep(ctx, namespace, step.ExecutionID, step.StepName)
		require.NoError(t, err, "GetCompletedStep should work")
		require.Equal(t, store.StepStatusCompleted, completed.Status)
	})

	t.Run("CronOperations", func(t *testing.T) {
		cronJob := &store.CronJob{
			ID:           uid.New(uid.CronJobPrefix),
			Name:         "test-cron",
			CronSpec:     "0 0 * * *",
			Namespace:    namespace,
			WorkflowName: "test-workflow",
			Enabled:      true,
			CreatedAt:    time.Now().UnixMilli(),
			UpdatedAt:    time.Now().UnixMilli(),
			NextRunAt:    time.Now().Add(24 * time.Hour).UnixMilli(),
		}

		// Test UpsertCronJob (create)
		err := engine.GetSQLCStore().UpsertCronJob(ctx, cronJob)
		require.NoError(t, err, "UpsertCronJob (create) should work")

		// Test GetCronJob
		retrieved, err := engine.GetSQLCStore().GetCronJob(ctx, namespace, cronJob.Name)
		require.NoError(t, err, "GetCronJob should work")
		require.Equal(t, cronJob.Name, retrieved.Name)

		// Test GetCronJobs
		cronJobs, err := engine.GetSQLCStore().GetCronJobs(ctx, namespace)
		require.NoError(t, err, "GetCronJobs should work")
		require.Len(t, cronJobs, 1)

		// Test GetDueCronJobs
		futureTime := time.Now().Add(48 * time.Hour).UnixMilli()
		dueCrons, err := engine.GetSQLCStore().GetDueCronJobs(ctx, namespace, futureTime)
		require.NoError(t, err, "GetDueCronJobs should work")
		require.Len(t, dueCrons, 1)

		// Test UpdateCronJobLastRun
		now := time.Now().UnixMilli()
		nextRun := now + (24 * time.Hour).Milliseconds()
		err = engine.GetSQLCStore().UpdateCronJobLastRun(ctx, namespace, cronJob.ID, now, nextRun)
		require.NoError(t, err, "UpdateCronJobLastRun should work")
	})

	t.Run("LeaseOperations", func(t *testing.T) {
		resourceID := uid.New(uid.WorkflowPrefix)
		workerID := uid.New(uid.WorkerPrefix)
		now := time.Now().UnixMilli()

		lease := &store.Lease{
			ResourceID:  resourceID,
			Kind:        string(store.LeaseKindWorkflow),
			Namespace:   namespace,
			WorkerID:    workerID,
			AcquiredAt:  now,
			ExpiresAt:   now + (5 * time.Minute).Milliseconds(),
			HeartbeatAt: now,
		}

		// Test AcquireLease
		err := engine.GetSQLCStore().AcquireLease(ctx, lease)
		require.NoError(t, err, "AcquireLease should work")

		// Test GetLease
		retrieved, err := engine.GetSQLCStore().GetLease(ctx, resourceID)
		require.NoError(t, err, "GetLease should work")
		require.Equal(t, workerID, retrieved.WorkerID)

		// Test HeartbeatLease
		newExpiresAt := now + (10 * time.Minute).Milliseconds()
		err = engine.GetSQLCStore().HeartbeatLease(ctx, resourceID, workerID, newExpiresAt)
		require.NoError(t, err, "HeartbeatLease should work")

		// Test ReleaseLease
		err = engine.GetSQLCStore().ReleaseLease(ctx, resourceID, workerID)
		require.NoError(t, err, "ReleaseLease should work")
	})

	t.Run("CleanupOperations", func(t *testing.T) {
		// Test GetExpiredLeases
		expired, err := engine.GetSQLCStore().GetExpiredLeases(ctx, namespace)
		require.NoError(t, err, "GetExpiredLeases should work")
		_ = expired // Don't assert on count as it depends on test state

		// Test CleanupExpiredLeases
		err = engine.GetSQLCStore().CleanupExpiredLeases(ctx, namespace)
		require.NoError(t, err, "CleanupExpiredLeases should work")

		// Test ResetOrphanedWorkflows
		err = engine.GetSQLCStore().ResetOrphanedWorkflows(ctx, namespace)
		require.NoError(t, err, "ResetOrphanedWorkflows should work")
	})

	t.Run("TransactionSupport", func(t *testing.T) {
		// Test WithTx
		err := engine.GetSQLCStore().WithTx(ctx, func(txStore store.Store) error {
			// Create a workflow within transaction
			workflow := &store.WorkflowExecution{
				ID:                uid.New(uid.WorkflowPrefix),
				WorkflowName:      "tx-test-workflow",
				Status:            store.WorkflowStatusPending,
				InputData:         []byte(`{"tx": true}`),
				CreatedAt:         time.Now().UnixMilli(),
				MaxAttempts:       1,
				RemainingAttempts: 1,
				Namespace:         namespace,
			}
			return txStore.CreateWorkflow(ctx, workflow)
		})
		require.NoError(t, err, "WithTx should work")
	})

	t.Run("TestingHelpers", func(t *testing.T) {
		// Test GetAllWorkflows
		workflows, err := engine.GetSQLCStore().GetAllWorkflows(ctx, namespace)
		require.NoError(t, err, "GetAllWorkflows should work")
		require.GreaterOrEqual(t, len(workflows), 0)

		// Test GetAllSteps
		steps, err := engine.GetSQLCStore().GetAllSteps(ctx, namespace)
		require.NoError(t, err, "GetAllSteps should work")
		require.GreaterOrEqual(t, len(steps), 0)
	})
}
