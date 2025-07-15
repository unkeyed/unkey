package orchestrator

import "context"

// StepFunc represents a function that executes a single step
type StepFunc func(ctx context.Context) error

// Step represents a single operation in a multi-step process
type Step struct {
	ID         string
	Name       string
	Execute    StepFunc
	OnSuccess  func() string      // Optional: custom success message
	OnError    func(error) string // Optional: custom error message
	SkipIf     func() bool        // Optional: condition to skip this step
	SkipReason func() string      // Optional: reason for skipping
	Required   bool               // If true, failure stops the entire process
}

// StepBuilder provides a fluent interface for building steps
type StepBuilder struct {
	step *Step
}

// NewStep creates a new step builder
func NewStep(id, name string) *StepBuilder {
	return &StepBuilder{
		step: &Step{
			ID:       id,
			Name:     name,
			Required: true, // Default to required
		},
	}
}

// Execute sets the execution function
func (sb *StepBuilder) Execute(fn StepFunc) *StepBuilder {
	sb.step.Execute = fn
	return sb
}

// OnSuccess sets the success message function
func (sb *StepBuilder) OnSuccess(fn func() string) *StepBuilder {
	sb.step.OnSuccess = fn
	return sb
}

// OnError sets the error message function
func (sb *StepBuilder) OnError(fn func(error) string) *StepBuilder {
	sb.step.OnError = fn
	return sb
}

// SkipIf sets the skip condition
func (sb *StepBuilder) SkipIf(fn func() bool) *StepBuilder {
	sb.step.SkipIf = fn
	return sb
}

// SkipReason sets the skip reason function
func (sb *StepBuilder) SkipReason(fn func() string) *StepBuilder {
	sb.step.SkipReason = fn
	return sb
}

// Required sets whether the step is required (default: true)
func (sb *StepBuilder) Required(required bool) *StepBuilder {
	sb.step.Required = required
	return sb
}

// Optional marks the step as optional (failure won't stop execution)
func (sb *StepBuilder) Optional() *StepBuilder {
	sb.step.Required = false
	return sb
}

// Build returns the constructed step
func (sb *StepBuilder) Build() *Step {
	return sb.step
}

// ConditionalStep creates a step that can be skipped based on a condition
func ConditionalStep(id, name string, fn StepFunc, skipIf func() bool, skipReason func() string) *Step {
	return NewStep(id, name).
		Execute(fn).
		SkipIf(skipIf).
		SkipReason(skipReason).
		Build()
}
