package hydra

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Sleep suspends workflow execution for the specified duration.
//
// This function allows workflows to pause execution and resume after
// a specified time period. The workflow will be marked as sleeping
// and workers will not attempt to execute it until the sleep duration
// has elapsed.
//
// Sleep is useful for:
// - Time-based coordination (e.g., waiting for settlement periods)
// - Human approval workflows (e.g., waiting for manual intervention)
// - Rate limiting and backoff strategies
// - Scheduled processing windows
//
// The sleep duration is durable - if the worker crashes or restarts
// during the sleep period, the workflow will still resume at the
// correct time.
//
// Example usage:
//
//	// Sleep for 24 hours for manual approval
//	err = hydra.Sleep(ctx, 24*time.Hour)
//	if err != nil {
//	    return err
//	}
//
//	// Continue with post-approval processing
//	result, err := hydra.Step(ctx, "post-approval", func(stepCtx context.Context) (string, error) {
//	    return processApprovedRequest(stepCtx)
//	})
//
// Note: Sleep creates an internal step to track the sleep state.
// The step name is generated automatically based on the duration.
//
// Metrics recorded:
// - hydra_sleeps_started_total (counter)
// - hydra_workflows_sleeping (gauge)
func Sleep(ctx WorkflowContext, duration time.Duration) error {
	wctx, ok := ctx.(*workflowContext)
	if !ok {
		return fmt.Errorf("invalid workflow context")
	}

	stepName := fmt.Sprintf("sleep-%d", duration.Milliseconds())

	_, err := store.Query.GetCompletedStep(wctx.ctx, wctx.db, store.GetCompletedStepParams{
		Namespace:   wctx.namespace,
		ExecutionID: wctx.ExecutionID(),
		StepName:    stepName,
	})
	if err == nil {
		return nil
	}

	now := time.Now().UnixMilli()
	existingStep, err := store.Query.GetStep(wctx.ctx, wctx.db, store.GetStepParams{
		Namespace:   wctx.namespace,
		ExecutionID: wctx.ExecutionID(),
		StepName:    stepName,
	})
	if err == nil && existingStep.StartedAt.Valid {
		sleepUntil := existingStep.StartedAt.Int64 + duration.Milliseconds()

		if sleepUntil <= now {
			return wctx.markStepCompleted(existingStep.ID, []byte("{}"))
		}
		return store.Query.SleepWorkflow(wctx.ctx, wctx.db, store.SleepWorkflowParams{
			SleepUntil: sql.NullInt64{Int64: sleepUntil, Valid: true},
			ID:         wctx.ExecutionID(),
			Namespace:  wctx.namespace,
		})
	}

	sleepUntil := now + duration.Milliseconds()

	// Create sleep step with lease validation - only if worker holds valid lease
	stepID := uid.New(uid.StepPrefix)
	result, err := wctx.db.ExecContext(wctx.ctx, `
		INSERT INTO workflow_steps (
		    id, execution_id, step_name, status, output_data, error_message,
		    started_at, completed_at, max_attempts, remaining_attempts, namespace
		) 
		SELECT ?, ?, ?, ?, ?, ?,
		       ?, ?, ?, ?, ?
		WHERE EXISTS (
		    SELECT 1 FROM leases 
		    WHERE resource_id = ? AND kind = 'workflow' 
		    AND worker_id = ? AND expires_at > ?
		)`,
		stepID,
		wctx.ExecutionID(),
		stepName,
		store.WorkflowStepsStatusRunning,
		[]byte{},
		sql.NullString{String: "", Valid: false},
		sql.NullInt64{Int64: now, Valid: true},
		sql.NullInt64{Int64: 0, Valid: false},
		1, // Sleep doesn't need retries
		1,
		wctx.namespace,
		wctx.ExecutionID(), // resource_id for lease check
		wctx.workerID,      // worker_id for lease check
		now,                // expires_at check
	)
	if err != nil {
		return fmt.Errorf("failed to create sleep step: %w", err)
	}

	// Check if the step creation actually happened (lease validation)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check step creation result: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("sleep step creation failed: lease expired or invalid")
	}

	return store.Query.SleepWorkflow(wctx.ctx, wctx.db, store.SleepWorkflowParams{
		SleepUntil: sql.NullInt64{Int64: sleepUntil, Valid: true},
		ID:         wctx.ExecutionID(),
		Namespace:  wctx.namespace,
	})
}
