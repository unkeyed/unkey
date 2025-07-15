// Package orchestrator provides a reusable pattern for executing multi-step operations
// with progress tracking and error handling.
package orchestrator

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/cmd/cli/progress"
)

// Orchestrator manages the execution of multiple steps with progress tracking
type Orchestrator struct {
	title   string
	steps   []*Step
	tracker *progress.Tracker
	ctx     context.Context
	state   map[string]any // Shared state between steps
}

// New creates a new orchestrator with the given title
func New(ctx context.Context, title string) *Orchestrator {
	return &Orchestrator{
		title: title,
		steps: make([]*Step, 0),
		ctx:   ctx,
		state: make(map[string]any),
	}
}

// AddStep adds a step to the orchestrator
func (o *Orchestrator) AddStep(step *Step) *Orchestrator {
	o.steps = append(o.steps, step)
	return o
}

// AddSteps adds multiple steps to the orchestrator
func (o *Orchestrator) AddSteps(steps ...*Step) *Orchestrator {
	for _, step := range steps {
		o.AddStep(step)
	}
	return o
}

// Execute runs all the steps in sequence with progress tracking
func (o *Orchestrator) Execute() error {
	// Initialize tracker
	o.tracker = progress.NewTracker(o.title)

	// Add all steps to tracker
	for _, step := range o.steps {
		o.tracker.AddStep(step.ID, step.Name)
	}

	o.tracker.Start()
	defer o.tracker.Stop()

	// Execute steps
	for _, step := range o.steps {
		if err := o.executeStep(step); err != nil {
			return err
		}
	}

	return nil
}

// executeStep executes a single step with proper error handling
func (o *Orchestrator) executeStep(step *Step) error {
	// Safety check for nil Execute function
	if step.Execute == nil {
		return fmt.Errorf("step '%s' has no execution function", step.Name)
	}

	// Check if step should be skipped
	if step.SkipIf != nil && step.SkipIf() {
		reason := fmt.Sprintf("Skipped: %s", step.Name)
		if step.SkipReason != nil {
			reason = step.SkipReason()
		}
		o.tracker.SkipStep(step.ID, reason)
		return nil
	}

	// Start the step
	o.tracker.StartStep(step.ID, fmt.Sprintf("Executing %s", step.Name))

	// Execute the step
	err := step.Execute(o.ctx)
	if err != nil {
		// Handle error
		errorMsg := err.Error()
		if step.OnError != nil {
			errorMsg = step.OnError(err)
		}

		o.tracker.FailStep(step.ID, errorMsg)

		// If step is required, stop execution
		if step.Required {
			return fmt.Errorf("required step '%s' failed: %w", step.Name, err)
		}

		// For non-required steps, continue execution
		return nil
	}

	// Handle success
	successMsg := fmt.Sprintf("%s completed", step.Name)
	if step.OnSuccess != nil {
		successMsg = step.OnSuccess()
	}

	o.tracker.CompleteStep(step.ID, successMsg)
	return nil
}

// UpdateStepMessage updates the message for a currently running step
func (o *Orchestrator) UpdateStepMessage(stepID, message string) {
	if o.tracker != nil {
		o.tracker.UpdateStep(stepID, message)
	}
}

// GetTracker returns the underlying progress tracker for advanced usage
// This should be used sparingly and only when the orchestrator pattern
// doesn't cover your specific use case
func (o *Orchestrator) GetTracker() *progress.Tracker {
	return o.tracker
}
