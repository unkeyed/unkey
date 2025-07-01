package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// ExampleWorkflow demonstrates how to create a workflow in the controlplane service.
// This is a simple example that engineers can use as a starting point.
type ExampleWorkflow struct {
	logger logging.Logger
}

// NewExampleWorkflow creates a new instance of the example workflow
func NewExampleWorkflow(logger logging.Logger) *ExampleWorkflow {
	return &ExampleWorkflow{
		logger: logger,
	}
}

// Name returns the unique name for this workflow type
func (w *ExampleWorkflow) Name() string {
	return "example-workflow"
}

// ExampleRequest represents the input data for the example workflow
type ExampleRequest struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

// Run executes the workflow logic
func (w *ExampleWorkflow) Run(ctx hydra.WorkflowContext, req ExampleRequest) error {
	w.logger.Info("example workflow started",
		"execution_id", ctx.ExecutionID(),
		"workflow_name", ctx.WorkflowName(),
		"message", req.Message,
		"user_id", req.UserID,
	)

	// Step 1: Validate input
	validationResult, err := hydra.Step(ctx, "validate-input", func(stepCtx context.Context) (string, error) {
		if req.Message == "" {
			return "", fmt.Errorf("message cannot be empty")
		}
		if req.UserID == "" {
			return "", fmt.Errorf("user_id cannot be empty")
		}

		w.logger.Info("input validation completed successfully")
		return "validation_passed", nil
	})
	if err != nil {
		w.logger.Error("input validation failed", "error", err)
		return fmt.Errorf("validation step failed: %w", err)
	}

	// Step 2: Process the message (simulate some business logic)
	processResult, err := hydra.Step(ctx, "process-message", func(stepCtx context.Context) (map[string]interface{}, error) {
		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)

		result := map[string]interface{}{
			"processed_message": fmt.Sprintf("Processed: %s", req.Message),
			"processed_at":      time.Now().UTC(),
			"validation_result": validationResult,
		}

		w.logger.Info("message processing completed",
			"processed_message", result["processed_message"],
			"user_id", req.UserID,
		)

		return result, nil
	})
	if err != nil {
		w.logger.Error("message processing failed", "error", err)
		return fmt.Errorf("processing step failed: %w", err)
	}

	// Step 3: Save result (simulate database save or external API call)
	saveResult, err := hydra.Step(ctx, "save-result", func(stepCtx context.Context) (string, error) {
		// Simulate saving to database or calling external service
		time.Sleep(50 * time.Millisecond)

		// In a real workflow, you would:
		// - Save to database
		// - Call external APIs
		// - Send notifications
		// - Update other systems

		w.logger.Info("result saved successfully",
			"user_id", req.UserID,
			"result", processResult,
		)

		return "save_completed", nil
	})
	if err != nil {
		w.logger.Error("save operation failed", "error", err)
		return fmt.Errorf("save step failed: %w", err)
	}

	w.logger.Info("example workflow completed successfully",
		"execution_id", ctx.ExecutionID(),
		"user_id", req.UserID,
		"validation_result", validationResult,
		"save_result", saveResult,
	)

	return nil
}

// Start is a convenience method to start this workflow
func (w *ExampleWorkflow) Start(ctx context.Context, engine *hydra.Engine, req ExampleRequest) (string, error) {
	return engine.StartWorkflow(ctx, w.Name(), req)
}

// Usage example:
//
// To register this workflow in your controlplane service:
//
// 1. In apps/controlplane/run.go, in the registerWorkflows function:
//
//    func registerWorkflows(worker hydra.Worker, logger logging.Logger) error {
//        exampleWorkflow := workflows.NewExampleWorkflow(logger)
//        err := hydra.RegisterWorkflow(worker, exampleWorkflow)
//        if err != nil {
//            return fmt.Errorf("unable to register example workflow: %w", err)
//        }
//
//        logger.Info("workflows registered successfully")
//        return nil
//    }
//
// 2. To start a workflow from another part of your application:
//
//    req := workflows.ExampleRequest{
//        Message: "Hello, World!",
//        UserID:  "user123",
//    }
//
//    workflowID, err := engine.StartWorkflow(ctx, "example-workflow", req)
//    if err != nil {
//        return fmt.Errorf("failed to start workflow: %w", err)
//    }
//
// 3. The workflow will be processed by any available controlplane worker and
//    will automatically handle retries, crash recovery, and exactly-once execution.
