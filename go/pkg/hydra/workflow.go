package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
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

// RawPayload represents raw workflow input data that needs to be unmarshalled
type RawPayload struct {
	Data []byte
}

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
	db              *sql.DB
	marshaller      Marshaller
	stepTimeout     time.Duration
	stepMaxAttempts int32
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

func (w *workflowContext) markStepCompleted(stepName string, outputData []byte) error {
	// Use simple step update - we're already in workflow execution context
	return store.Query.UpdateStepStatus(w.ctx, w.db, store.UpdateStepStatusParams{
		Status:       store.WorkflowStepsStatusCompleted,
		CompletedAt:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		OutputData:   outputData,
		ErrorMessage: sql.NullString{String: "", Valid: false},
		Namespace:    w.namespace,
		ExecutionID:  w.executionID,
		StepName:     stepName,
	})
}

func (w *workflowContext) markStepFailed(stepName string, errorMsg string) error {
	// Use simple step update - we're already in workflow execution context
	return store.Query.UpdateStepStatus(w.ctx, w.db, store.UpdateStepStatusParams{
		Status:       store.WorkflowStepsStatusFailed,
		CompletedAt:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		OutputData:   []byte{},
		ErrorMessage: sql.NullString{String: errorMsg, Valid: errorMsg != ""},
		Namespace:    w.namespace,
		ExecutionID:  w.executionID,
		StepName:     stepName,
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
	wctx, ok := ctx.(*workflowContext)
	if !ok {
		return fmt.Errorf("invalid context type, expected *workflowContext")
	}

	// Start tracing span for workflow execution
	workflowCtx, span := tracing.Start(wctx.ctx, fmt.Sprintf("hydra.workflow.%s", w.wrapped.Name()))
	defer span.End()

	span.SetAttributes(
		attribute.String("hydra.workflow.name", w.wrapped.Name()),
		attribute.String("hydra.execution.id", wctx.executionID),
		attribute.String("hydra.namespace", wctx.namespace),
		attribute.String("hydra.worker.id", wctx.workerID),
	)

	// Update the workflow context to use the traced context
	wctx.ctx = workflowCtx

	// Extract the raw payload and unmarshal it to the correct type
	rawPayload, ok := req.(*RawPayload)
	if !ok {
		err := fmt.Errorf("expected RawPayload, got %T", req)
		tracing.RecordError(span, err)
		return err
	}

	var typedReq TReq
	if err := wctx.marshaller.Unmarshal(rawPayload.Data, &typedReq); err != nil {
		tracing.RecordError(span, err)
		return fmt.Errorf("failed to unmarshal workflow request: %w", err)
	}

	// Pass the updated workflow context (with traced context) to the workflow implementation
	err := w.wrapped.Run(wctx, typedReq)
	if err != nil {
		tracing.RecordError(span, err)

		span.SetAttributes(attribute.String("hydra.workflow.status", "failed"))
	} else {
		span.SetAttributes(attribute.String("hydra.workflow.status", "completed"))
	}

	return err
}

// WorkflowOption defines a function that configures workflow execution
type WorkflowOption func(*WorkflowConfig)

// WorkflowConfig holds the configuration for workflow execution
type WorkflowConfig struct {
	MaxAttempts int32

	TimeoutDuration time.Duration

	RetryBackoff time.Duration

	TriggerType   store.WorkflowExecutionsTriggerType
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
func WithTrigger(triggerType store.WorkflowExecutionsTriggerType, triggerSource *string) WorkflowOption {
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
