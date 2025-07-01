package hydra

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/unkeyed/unkey/go/pkg/ptr"
)

func Step[TResponse any](ctx WorkflowContext, stepName string, fn func(context.Context) (TResponse, error)) (TResponse, error) {
	var zero TResponse

	wctx, ok := ctx.(*workflowContext)
	if !ok {
		return zero, fmt.Errorf("invalid workflow context")
	}

	existing, err := wctx.getCompletedStep(stepName)
	if err == nil && existing != nil {
		responseType := reflect.TypeOf((*TResponse)(nil)).Elem()
		var response TResponse

		if responseType.Kind() == reflect.Ptr {
			responseValue := reflect.New(responseType.Elem())
			response = responseValue.Interface().(TResponse)
		}

		if len(existing.OutputData) > 0 {
			err = wctx.marshaller.Unmarshal(existing.OutputData, &response)
			if err != nil {
				return zero, fmt.Errorf("failed to unmarshal cached step result: %w", err)
			}
		}

		return response, nil
	}

	existingStep, err := wctx.getAnyStep(stepName)
	var stepToUse *WorkflowStep
	var shouldCreateNewStep bool = true

	if err == nil && existingStep != nil {
		stepToUse = existingStep
		shouldCreateNewStep = false
	}

	if shouldCreateNewStep {
		stepToUse = &WorkflowStep{
			ExecutionID:       wctx.ExecutionID(),
			StepName:          stepName,
			StepOrder:         wctx.getNextStepOrder(),
			Status:            StepStatusRunning,
			Namespace:         wctx.namespace,
			StartedAt:         ptr.P(time.Now().UnixMilli()),
			MaxAttempts:       wctx.stepMaxAttempts,
			RemainingAttempts: wctx.stepMaxAttempts,
		}

		err = wctx.store.CreateStep(wctx.ctx, stepToUse)
		if err != nil {
			return zero, fmt.Errorf("failed to create step: %w", err)
		}
	} else {
		now := time.Now().UnixMilli()
		stepToUse.Status = StepStatusRunning
		stepToUse.StartedAt = ptr.P(now)

		stepToUse.ErrorMessage = ""
		stepToUse.CompletedAt = nil

		err = wctx.store.UpdateStepStatus(wctx.ctx, wctx.namespace, wctx.executionID, stepName, StepStatusRunning, nil, "")
		if err != nil {
			return zero, fmt.Errorf("failed to update step: %w", err)
		}
	}

	response, err := fn(wctx.ctx)
	if err != nil {
		wctx.markStepFailed(stepName, err.Error())
		return zero, fmt.Errorf("step execution failed: %w", err)
	}

	respData, err := wctx.marshaller.Marshal(response)
	if err != nil {
		wctx.markStepFailed(stepName, err.Error())
		return zero, fmt.Errorf("failed to marshal response: %w", err)
	}

	err = wctx.markStepCompleted(stepName, respData)
	if err != nil {
		return zero, fmt.Errorf("failed to mark step completed: %w", err)
	}

	return response, nil
}
