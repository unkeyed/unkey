package hydra

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/store"
)

// Workflow defines the interface for typed workflows
type Workflow[TReq any] interface {
	Name() string
	Run(ctx WorkflowContext, req TReq) error
}

// GenericWorkflow is a type alias for workflows that accept any request type
type GenericWorkflow = Workflow[any]

// WorkflowContext provides access to workflow execution context and utilities
type WorkflowContext interface {
	Context() context.Context
	ExecutionID() string
	WorkflowName() string
}

// workflowContext implements WorkflowContext and provides internal workflow utilities
type workflowContext struct {
	ctx             context.Context
	executionID     string
	workflowName    string
	namespace       string
	workerID        string
	store           store.Store
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
	return int32(w.stepOrder)
}

func (w *workflowContext) getCompletedStep(stepName string) (*store.WorkflowStep, error) {
	return w.store.GetCompletedStep(w.ctx, w.namespace, w.executionID, stepName)
}

func (w *workflowContext) getAnyStep(stepName string) (*store.WorkflowStep, error) {
	return w.store.GetStep(w.ctx, w.namespace, w.executionID, stepName)
}

func (w *workflowContext) markStepCompleted(stepName string, outputData []byte) error {
	return w.store.UpdateStepStatus(w.ctx, w.namespace, w.executionID, stepName, store.StepStatusCompleted, outputData, "")
}

func (w *workflowContext) markStepFailed(stepName string, errorMsg string) error {
	return w.store.UpdateStepStatus(w.ctx, w.namespace, w.executionID, stepName, store.StepStatusFailed, nil, errorMsg)
}

func (w *workflowContext) suspendWorkflowForSleep(sleepUntil int64) error {
	return w.store.SleepWorkflow(w.ctx, w.namespace, w.executionID, sleepUntil)
}

// RegisterWorkflow registers a typed workflow with a worker, handling the type conversion transparently
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
	wctx := ctx.(*workflowContext)
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

// WorkflowSuspendedError represents an error that suspends workflow execution until a specific time
type WorkflowSuspendedError struct {
	Reason string

	ResumeTime int64
}

func (e *WorkflowSuspendedError) Error() string {
	return fmt.Sprintf("workflow suspended for %s until %d", e.Reason, e.ResumeTime)
}
