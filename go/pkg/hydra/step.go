package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/metrics"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"go.opentelemetry.io/otel/attribute"
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

	// Start tracing span for this step
	stepCtx, span := tracing.Start(wctx.ctx, fmt.Sprintf("hydra.step.%s", stepName))
	defer span.End()

	span.SetAttributes(
		attribute.String("hydra.step.name", stepName),
		attribute.String("hydra.workflow.name", wctx.workflowName),
		attribute.String("hydra.execution.id", wctx.executionID),
		attribute.String("hydra.namespace", wctx.namespace),
	)

	existing, err := store.Query.GetCompletedStep(wctx.ctx, wctx.db, store.GetCompletedStepParams{
		Namespace:   wctx.namespace,
		ExecutionID: wctx.ExecutionID(),
		StepName:    stepName,
	})
	if err == nil {
		// Record cached step hit
		metrics.StepsCachedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName).Inc()
		span.SetAttributes(attribute.Bool("hydra.step.cached", true))

		responseType := reflect.TypeOf((*TResponse)(nil)).Elem()
		var response TResponse

		if responseType.Kind() == reflect.Ptr {
			responseValue := reflect.New(responseType.Elem())
			var ok bool
			response, ok = responseValue.Interface().(TResponse)
			if !ok {
				conversionErr := fmt.Errorf("failed to convert response to expected type")
				tracing.RecordError(span, conversionErr)
				return zero, conversionErr
			}
		}

		if len(existing.OutputData) > 0 {
			err = wctx.marshaller.Unmarshal(existing.OutputData, &response)
			if err != nil {
				metrics.RecordError(wctx.namespace, "step", "unmarshal_cached_result_failed")
				tracing.RecordError(span, err)
				return zero, fmt.Errorf("failed to unmarshal cached step result: %w", err)
			}
		}

		return response, nil
	}

	span.SetAttributes(attribute.Bool("hydra.step.cached", false))

	_, err = store.Query.GetStep(wctx.ctx, wctx.db, store.GetStepParams{
		Namespace:   wctx.namespace,
		ExecutionID: wctx.ExecutionID(),
		StepName:    stepName,
	})
	var stepToUse *store.WorkflowStep
	shouldCreateNewStep := err != nil // sql.ErrNoRows means step doesn't exist, so create new one

	stepStartTime := time.Now()

	if shouldCreateNewStep {
		stepID := uid.New(uid.StepPrefix)

		// Create step with lease validation - only if worker holds valid lease
		now := time.Now().UnixMilli()
		createResult, createErr := wctx.db.ExecContext(wctx.ctx, `
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
			sql.NullInt64{Int64: stepStartTime.UnixMilli(), Valid: true},
			sql.NullInt64{Int64: 0, Valid: false},
			wctx.stepMaxAttempts,
			wctx.stepMaxAttempts,
			wctx.namespace,
			wctx.ExecutionID(), // resource_id for lease check
			wctx.workerID,      // worker_id for lease check
			now,                // expires_at check
		)
		if createErr != nil {
			return zero, fmt.Errorf("failed to create step: %w", createErr)
		}

		// Check if the step creation actually happened (lease validation)
		rowsAffected, rowsErr := createResult.RowsAffected()
		if rowsErr != nil {
			return zero, fmt.Errorf("failed to check step creation result: %w", rowsErr)
		}
		if rowsAffected == 0 {
			return zero, fmt.Errorf("step creation failed: lease expired or invalid")
		}

		// Step created successfully
		span.SetAttributes(attribute.Bool("hydra.step.new", true))
	} else {
		// Update existing step to running status with lease validation
		now := time.Now().UnixMilli()
		updateResult, updateErr := wctx.db.ExecContext(wctx.ctx, `
			UPDATE workflow_steps
			SET status = ?, completed_at = ?, output_data = ?, error_message = ?
			WHERE workflow_steps.namespace = ? AND execution_id = ? AND step_name = ?
			  AND EXISTS (
			    SELECT 1 FROM leases
			    WHERE resource_id = ? AND kind = 'workflow'
			    AND worker_id = ? AND expires_at > ?
			  )`,
			store.WorkflowStepsStatusRunning,
			sql.NullInt64{Int64: 0, Valid: false},
			[]byte{},
			sql.NullString{String: "", Valid: false},
			wctx.namespace,
			wctx.ExecutionID(),
			stepName,
			wctx.ExecutionID(), // resource_id for lease check
			wctx.workerID,      // worker_id for lease check
			now,                // expires_at check
		)
		if updateErr != nil {
			return zero, fmt.Errorf("failed to update step status: %w", updateErr)
		}

		// Check if the update actually happened (lease validation)
		rowsAffected, rowsErr := updateResult.RowsAffected()
		if rowsErr != nil {
			return zero, fmt.Errorf("failed to check step update result: %w", rowsErr)
		}
		if rowsAffected == 0 {
			return zero, fmt.Errorf("step update failed: lease expired or invalid")
		}

		// Get the step after successful update
		stepResult, getErr := store.Query.GetStep(wctx.ctx, wctx.db, store.GetStepParams{
			Namespace:   wctx.namespace,
			ExecutionID: wctx.ExecutionID(),
			StepName:    stepName,
		})
		stepToUse = &stepResult
		if getErr != nil {
			return zero, fmt.Errorf("failed to retrieve updated step: %w", getErr)
		}

		// Record step retry
		if stepToUse.RemainingAttempts < stepToUse.MaxAttempts {
			metrics.StepsRetriedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName).Inc()
			span.SetAttributes(attribute.Bool("hydra.step.retry", true))
		}
		span.SetAttributes(attribute.Bool("hydra.step.new", false))
	}

	response, err := fn(stepCtx)
	if err != nil {
		tracing.RecordError(span, err)
		wctx.logger.Error("step execution failed", "error", err.Error())
		span.SetAttributes(attribute.String("hydra.step.status", "failed"))

		if markErr := wctx.markStepFailed(stepName, err.Error()); markErr != nil {
			metrics.RecordError(wctx.namespace, "step", "mark_step_failed_error")
		}
		metrics.ObserveStepDuration(wctx.namespace, wctx.workflowName, stepName, "failed", stepStartTime)
		metrics.StepsExecutedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName, "failed").Inc()
		return zero, fmt.Errorf("step execution failed: %w", err)
	}

	respData, err := wctx.marshaller.Marshal(response)
	if err != nil {
		tracing.RecordError(span, err)
		span.SetAttributes(attribute.String("hydra.step.status", "failed"))

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
		tracing.RecordError(span, err)
		span.SetAttributes(attribute.String("hydra.step.status", "failed"))
		metrics.RecordError(wctx.namespace, "step", "mark_completed_failed")
		return zero, fmt.Errorf("failed to mark step completed: %w", err)
	}

	span.SetAttributes(attribute.String("hydra.step.status", "completed"))
	metrics.ObserveStepDuration(wctx.namespace, wctx.workflowName, stepName, "completed", stepStartTime)
	metrics.StepsExecutedTotal.WithLabelValues(wctx.namespace, wctx.workflowName, stepName, "completed").Inc()

	return response, nil
}

// StepVoid executes a named step within a workflow that performs side effects but doesn't return a value.
//
// This is a convenience wrapper around Step for functions that only return an error.
// It's perfect for steps that perform database updates, send notifications, or other
// side effects where the result itself isn't needed by subsequent steps.
//
// Parameters:
// - ctx: The workflow context from the workflow's Run() method
// - stepName: A unique name for this step within the workflow
// - fn: The function to execute, which should be idempotent and only return an error
//
// Example usage:
//
//	// Database update step
//	err := hydra.StepVoid(ctx, "update-user-status", func(stepCtx context.Context) error {
//	    return userService.UpdateStatus(stepCtx, userID, "active")
//	})
//
//	// Notification step
//	err := hydra.StepVoid(ctx, "send-email", func(stepCtx context.Context) error {
//	    return emailService.SendWelcomeEmail(stepCtx, userEmail)
//	})
//
// Returns only an error if the step execution fails.
func StepVoid(ctx WorkflowContext, stepName string, fn func(context.Context) error) error {
	_, err := Step(ctx, stepName, func(stepCtx context.Context) (*struct{}, error) {
		if err := fn(stepCtx); err != nil {
			return nil, err
		}
		return &struct{}{}, nil
	})
	return err
}
