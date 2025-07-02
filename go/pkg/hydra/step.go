package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/db"
	"github.com/unkeyed/unkey/go/pkg/hydra/metrics"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Step executes a named step within a workflow with automatic checkpointing and retry logic.
//
// Steps are the fundamental units of work in Hydra workflows. They provide:
// - Exactly-once execution guarantees
// - Automatic result caching (checkpointing)
// - Built-in retry logic for transient failures
// - Comprehensive metrics and observability
//
// Parameters:
// - ctx: The workflow context from the workflow's Run() method
// - stepName: A unique name for this step within the workflow
// - fn: The function to execute, which should be idempotent
//
// The stepName must be unique within the workflow and should remain stable
// across deployments. If a step has already completed successfully, its
// cached result will be returned without re-executing the function.
//
// The function fn receives a standard Go context and should:
// - Be idempotent (safe to run multiple times)
// - Handle context cancellation gracefully
// - Return consistent results for the same inputs
// - Use the provided context for any I/O operations
//
// Example usage:
//
//	// Simple step with string result
//	result, err := hydra.Step(ctx, "fetch-user", func(stepCtx context.Context) (string, error) {
//	    user, err := userService.GetUser(stepCtx, userID)
//	    if err != nil {
//	        return "", err
//	    }
//	    return user.Name, nil
//	})
//
//	// Step with complex result type
//	order, err := hydra.Step(ctx, "create-order", func(stepCtx context.Context) (*Order, error) {
//	    return orderService.CreateOrder(stepCtx, &CreateOrderRequest{
//	        CustomerID: customerID,
//	        Items:      items,
//	    })
//	})
//
// Metrics recorded:
// - hydra_steps_executed_total (counter with status)
// - hydra_step_duration_seconds (histogram)
// - hydra_steps_cached_total (counter for cache hits)
// - hydra_steps_retried_total (counter for retry attempts)
//
// Returns the result of the function execution or the cached result if the
// step has already completed successfully.
func Step[TResponse any](ctx WorkflowContext, stepName string, fn func(context.Context) (TResponse, error)) (TResponse, error) {
	var zero TResponse

	wctx, ok := ctx.(*workflowContext)
	if !ok {
		return zero, fmt.Errorf("invalid workflow context")
	}

	existing, err := wctx.getCompletedStep(stepName)
	if err == nil && existing != nil {
		// Record cached step hit
		metrics.StepsCachedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName).Inc()

		responseType := reflect.TypeOf((*TResponse)(nil)).Elem()
		var response TResponse

		if responseType.Kind() == reflect.Ptr {
			responseValue := reflect.New(responseType.Elem())
			var ok bool
			response, ok = responseValue.Interface().(TResponse)
			if !ok {
				return zero, fmt.Errorf("failed to convert response to expected type")
			}
		}

		if len(existing.OutputData) > 0 {
			err = wctx.marshaller.Unmarshal(existing.OutputData, &response)
			if err != nil {
				metrics.RecordError(wctx.namespace, "step", "unmarshal_cached_result_failed")
				return zero, fmt.Errorf("failed to unmarshal cached step result: %w", err)
			}
		}

		return response, nil
	}

	existingStep, err := wctx.getAnyStep(stepName)
	var stepToUse *db.WorkflowStep
	shouldCreateNewStep := true

	if err == nil && existingStep != nil {
		stepToUse = existingStep
		shouldCreateNewStep = false
	}

	stepStartTime := time.Now()

	if shouldCreateNewStep {
		stepID := uid.New("step")
		err = db.Query.CreateStep(wctx.ctx, wctx.db, db.CreateStepParams{
			ID:                stepID,
			ExecutionID:       wctx.ExecutionID(),
			StepName:          stepName,
			StepOrder:         wctx.getNextStepOrder(),
			Status:            db.WorkflowStepsStatusRunning,
			Namespace:         wctx.namespace,
			MaxAttempts:       wctx.stepMaxAttempts,
			RemainingAttempts: wctx.stepMaxAttempts,
		})
		if err != nil {
			metrics.RecordError(wctx.namespace, "step", "create_step_failed")
			return zero, fmt.Errorf("failed to create step: %w", err)
		}
	} else {
		err = db.Query.UpdateStepStatus(wctx.ctx, wctx.db, db.UpdateStepStatusParams{
			Status:       db.WorkflowStepsStatusRunning,
			CompletedAt:  sql.NullInt64{},
			OutputData:   nil,
			ErrorMessage: sql.NullString{},
			Namespace:    wctx.namespace,
			ExecutionID:  wctx.executionID,
			StepName:     stepName,
		})
		if err != nil {
			metrics.RecordError(wctx.namespace, "step", "update_step_failed")
			return zero, fmt.Errorf("failed to update step: %w", err)
		}

		// Record step retry
		if stepToUse.RemainingAttempts < stepToUse.MaxAttempts {
			metrics.StepsRetriedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName).Inc()
		}
	}

	response, err := fn(wctx.ctx)
	if err != nil {
		if markErr := wctx.markStepFailed(stepName, err.Error()); markErr != nil {
			metrics.RecordError(wctx.namespace, "step", "mark_step_failed_error")
		}
		metrics.ObserveStepDuration(wctx.namespace, wctx.workflowName, stepName, "failed", stepStartTime)
		metrics.StepsExecutedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName, "failed").Inc()
		return zero, fmt.Errorf("step execution failed: %w", err)
	}

	respData, err := wctx.marshaller.Marshal(response)
	if err != nil {
		if markErr := wctx.markStepFailed(stepName, err.Error()); markErr != nil {
			metrics.RecordError(wctx.namespace, "step", "mark_step_failed_error")
		}
		metrics.ObserveStepDuration(wctx.namespace, wctx.workflowName, stepName, "failed", stepStartTime)
		metrics.StepsExecutedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName, "failed").Inc()
		metrics.RecordError(wctx.namespace, "step", "marshal_response_failed")
		return zero, fmt.Errorf("failed to marshal response: %w", err)
	}

	err = wctx.markStepCompleted(stepName, respData)
	if err != nil {
		metrics.RecordError(wctx.namespace, "step", "mark_completed_failed")
		return zero, fmt.Errorf("failed to mark step completed: %w", err)
	}

	metrics.ObserveStepDuration(wctx.namespace, wctx.workflowName, stepName, "completed", stepStartTime)
	metrics.StepsExecutedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName, "completed").Inc()

	return response, nil
}
