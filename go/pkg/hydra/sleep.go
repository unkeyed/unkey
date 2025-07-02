package hydra

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/db"
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

	existing, err := wctx.getCompletedStep(stepName)
	if err == nil && existing != nil {
		return nil
	}

	now := time.Now().UnixMilli()
	existingStep, err := wctx.getAnyStep(stepName)
	if err == nil && existingStep != nil && existingStep.StartedAt.Valid {
		sleepUntil := existingStep.StartedAt.Int64 + duration.Milliseconds()

		if sleepUntil <= now {
			return wctx.markStepCompleted(stepName, []byte("{}"))
		}
		return wctx.suspendWorkflowForSleep(sleepUntil)
	}

	sleepUntil := now + duration.Milliseconds()

	err = db.Query.CreateStep(wctx.ctx, wctx.db, db.CreateStepParams{
		ID:                uid.New("step"),
		ExecutionID:       wctx.ExecutionID(),
		StepName:          stepName,
		StepOrder:         wctx.getNextStepOrder(),
		Status:            db.WorkflowStepsStatusRunning,
		Namespace:         wctx.namespace,
		MaxAttempts:       1, // Sleep doesn't need retries
		RemainingAttempts: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to create sleep step: %w", err)
	}

	return wctx.suspendWorkflowForSleep(sleepUntil)
}
