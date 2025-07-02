package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/db"
)

// Workflow defines the interface for typed workflows.
//
// Workflows are the core business logic containers in Hydra. They define
// a series of steps to be executed reliably with exactly-once guarantees.
//
// Workflows must be stateless and deterministic - they can be executed
// multiple times with the same input and produce the same result. State
// is managed by the workflow engine and persisted automatically.
//
// Type parameter TReq defines the input payload type for the workflow.
// Use 'any' for workflows that accept different payload types.
//
// Example implementation:
//
//	type OrderWorkflow struct{}
//
//	func (w *OrderWorkflow) Name() string {
//	    return "order-processing"
//	}
//
//	func (w *OrderWorkflow) Run(ctx hydra.WorkflowContext, req *OrderRequest) error {
//	    // Execute steps using hydra.Step()
//	    payment, err := hydra.Step(ctx, "validate-payment", func(stepCtx context.Context) (*Payment, error) {
//	        return validatePayment(stepCtx, req.PaymentID)
//	    })
//	    if err != nil {
//	        return err
//	    }
//
//	    // Additional steps...
//	    return nil
//	}
type Workflow[TReq any] interface {
	// Name returns a unique identifier for this workflow type.
	// The name is used to route workflow executions to the correct handler
	// and must be consistent across deployments.
	Name() string

	// Run executes the workflow logic with the provided context and request.
	// This method should be deterministic and idempotent.
	//
	// The context provides access to workflow execution metadata and
	// the Step() function for creating durable execution units.
	//
	// Returning an error will mark the workflow as failed and trigger
	// retry logic if configured. Use hydra.Sleep() to suspend the
	// workflow for time-based coordination.
	Run(ctx WorkflowContext, req TReq) error
}

// GenericWorkflow is a type alias for workflows that accept any request type.
// This is useful when registering workflows that handle different payload types
// or when the payload type is not known at compile time.
type GenericWorkflow = Workflow[any]

// WorkflowContext provides access to workflow execution context and utilities.
//
// The context is passed to workflow Run() methods and provides access to:
// - The underlying Go context for cancellation and timeouts
// - Workflow execution metadata like execution ID and name
// - Step execution utilities through the Step() function
//
// Workflow contexts are created and managed by the workflow engine and
// should not be created manually.
type WorkflowContext interface {
	// Context returns the underlying Go context for this workflow execution.
	// This context will be cancelled if the workflow is cancelled or times out.
	Context() context.Context

	// ExecutionID returns the unique identifier for this workflow execution.
	// This ID can be used for logging, tracking, and debugging purposes.
	ExecutionID() string

	// WorkflowName returns the name of the workflow being executed.
	// This matches the value returned by the workflow's Name() method.
	WorkflowName() string
}

// workflowContext implements WorkflowContext and provides internal workflow utilities
type workflowContext struct {
	ctx             context.Context
	executionID     string
	workflowName    string
	namespace       string
	workerID        string
	db              db.DBTX
	marshaller      Marshaller
	stepTimeout     time.Duration
	stepMaxAttempts int32
	stepOrder       int
}

func (w *workflowContext) Context() context.Context {
	return w.ctx
}

func (w *workflowContext) ExecutionID() string {
	return w.executionID
}

func (w *workflowContext) WorkflowName() string {
	return w.workflowName
}

func (w *workflowContext) getNextStepOrder() int32 {
	w.stepOrder++
	return int32(w.stepOrder) // nolint:gosec // Overflow is extremely unlikely in practice
}

func (w *workflowContext) getCompletedStep(stepName string) (*db.WorkflowStep, error) {
	step, err := db.Query.GetStep(w.ctx, w.db, db.GetStepParams{
		Namespace:   w.namespace,
		ExecutionID: w.executionID,
		StepName:    stepName,
	})
	if err != nil {
		return nil, err
	}

	// Check if step is completed
	if step.Status != db.WorkflowStepsStatusCompleted {
		return nil, sql.ErrNoRows // Return same error as if step wasn't found
	}

	return &step, nil
}

func (w *workflowContext) getAnyStep(stepName string) (*db.WorkflowStep, error) {
	step, err := db.Query.GetStep(w.ctx, w.db, db.GetStepParams{
		Namespace:   w.namespace,
		ExecutionID: w.executionID,
		StepName:    stepName,
	})
	if err != nil {
		return nil, err
	}
	return &step, nil
}

func (w *workflowContext) markStepCompleted(stepName string, outputData []byte) error {
	return db.Query.UpdateStepStatus(w.ctx, w.db, db.UpdateStepStatusParams{
		Status:       db.WorkflowStepsStatusCompleted,
		CompletedAt:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		OutputData:   outputData,
		ErrorMessage: sql.NullString{},
		Namespace:    w.namespace,
		ExecutionID:  w.executionID,
		StepName:     stepName,
	})
}

func (w *workflowContext) markStepFailed(stepName string, errorMsg string) error {
	return db.Query.UpdateStepStatus(w.ctx, w.db, db.UpdateStepStatusParams{
		Status:       db.WorkflowStepsStatusFailed,
		CompletedAt:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		OutputData:   nil,
		ErrorMessage: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
		Namespace:    w.namespace,
		ExecutionID:  w.executionID,
		StepName:     stepName,
	})
}

func (w *workflowContext) suspendWorkflowForSleep(sleepUntil int64) error {
	return db.Query.SleepWorkflow(w.ctx, w.db, db.SleepWorkflowParams{
		SleepUntil: sql.NullInt64{Int64: sleepUntil, Valid: true},
		ID:         w.executionID,
		Namespace:  w.namespace,
	})
}

// RegisterWorkflow registers a typed workflow with a worker.
//
// This function associates a workflow implementation with a worker so that
// the worker can execute workflows of this type. The workflow's Name() method
// is used as the unique identifier for routing workflow executions.
//
// The function handles type conversion transparently, allowing strongly-typed
// workflow implementations to be registered with the generic worker interface.
//
// Parameters:
// - w: The worker that will execute this workflow type
// - workflow: The workflow implementation to register
//
// Example:
//
//	type OrderWorkflow struct{}
//
//	func (w *OrderWorkflow) Name() string { return "order-processing" }
//	func (w *OrderWorkflow) Run(ctx hydra.WorkflowContext, req *OrderRequest) error {
//	    // workflow implementation
//	    return nil
//	}
//
//	orderWorkflow := &OrderWorkflow{}
//	err := hydra.RegisterWorkflow(worker, orderWorkflow)
//	if err != nil {
//	    return err
//	}
//
// Requirements:
// - The workflow name must be unique within the worker
// - The workflow must implement the Workflow[TReq] interface
// - The worker must be started with Start() after registration
//
// Returns an error if:
// - A workflow with the same name is already registered
// - The worker type is invalid
func RegisterWorkflow[TReq any](w Worker, workflow Workflow[TReq]) error {
	worker, ok := w.(*worker)
	if !ok {
		return fmt.Errorf("invalid worker type")
	}

	if _, exists := worker.workflows[workflow.Name()]; exists {
		return fmt.Errorf("workflow %q is already registered", workflow.Name())
	}

	// Create a wrapper that handles the type conversion
	genericWorkflow := &workflowWrapper[TReq]{
		wrapped: workflow,
	}

	worker.workflows[workflow.Name()] = genericWorkflow
	return nil
}

// workflowWrapper wraps a typed workflow to implement GenericWorkflow
type workflowWrapper[TReq any] struct {
	wrapped Workflow[TReq]
}

func (w *workflowWrapper[TReq]) Name() string {
	return w.wrapped.Name()
}

func (w *workflowWrapper[TReq]) Run(ctx WorkflowContext, req any) error {
	// Extract the raw payload and unmarshal it to the correct type
	rawPayload, ok := req.(*RawPayload)
	if !ok {
		return fmt.Errorf("expected RawPayload, got %T", req)
	}

	var typedReq TReq
	wctx, ok := ctx.(*workflowContext)
	if !ok {
		return fmt.Errorf("invalid context type, expected *workflowContext")
	}
	if err := wctx.marshaller.Unmarshal(rawPayload.Data, &typedReq); err != nil {
		return fmt.Errorf("failed to unmarshal workflow request: %w", err)
	}

	return w.wrapped.Run(ctx, typedReq)
}

// WorkflowOption defines a function that configures workflow execution
type WorkflowOption func(*WorkflowConfig)

// WorkflowConfig holds the configuration for workflow execution
type WorkflowConfig struct {
	MaxAttempts int32

	TimeoutDuration time.Duration

	RetryBackoff time.Duration

	TriggerType   TriggerType
	TriggerSource *string
}

// WithMaxAttempts sets the maximum number of retry attempts for a workflow
func WithMaxAttempts(attempts int32) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.MaxAttempts = attempts
	}
}

// WithTimeout sets the timeout duration for a workflow
func WithTimeout(timeout time.Duration) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.TimeoutDuration = timeout
	}
}

// WithRetryBackoff sets the retry backoff duration for a workflow
func WithRetryBackoff(backoff time.Duration) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.RetryBackoff = backoff
	}
}

// WithTrigger sets the trigger type and source for a workflow
func WithTrigger(triggerType TriggerType, triggerSource *string) WorkflowOption {
	return func(c *WorkflowConfig) {
		c.TriggerType = triggerType
		c.TriggerSource = triggerSource
	}
}

// WorkflowSuspendedError represents an error that suspends workflow execution until a specific time
type WorkflowSuspendedError struct {
	Reason string

	ResumeTime int64
}

func (e *WorkflowSuspendedError) Error() string {
	return fmt.Sprintf("workflow suspended for %s until %d", e.Reason, e.ResumeTime)
}
