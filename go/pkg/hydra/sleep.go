package hydra

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/ptr"
)

// Sleep suspends workflow execution for the specified duration
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
	if err == nil && existingStep != nil && existingStep.StartedAt != nil {
		sleepUntil := *existingStep.StartedAt + duration.Milliseconds()

		if sleepUntil <= now {
			return wctx.markStepCompleted(existingStep.ID, []byte("{}"))
		}
		return wctx.suspendWorkflowForSleep(sleepUntil)
	}

	sleepUntil := now + duration.Milliseconds()

	step := &WorkflowStep{
		ExecutionID:       wctx.ExecutionID(),
		StepName:          stepName,
		StepOrder:         wctx.getNextStepOrder(),
		Status:            StepStatusRunning,
		Namespace:         wctx.namespace,
		StartedAt:         ptr.P(now),
		MaxAttempts:       1, // Sleep doesn't need retries
		RemainingAttempts: 1,
	}

	err = wctx.store.CreateStep(wctx.ctx, step)
	if err != nil {
		return fmt.Errorf("failed to create sleep step: %w", err)
	}

	return wctx.suspendWorkflowForSleep(sleepUntil)
}
