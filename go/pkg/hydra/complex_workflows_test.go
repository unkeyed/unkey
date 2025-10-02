package hydra

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// ComplexBillingWorkflow simulates a realistic billing workflow with multiple steps,
// error handling, retries, and conditional logic
type ComplexBillingWorkflow struct {
	engine       *Engine
	name         string
	failureRate  float64 // Probability of step failure (0.0-1.0)
	chaosEnabled bool
	metrics      *WorkflowMetrics
}

// WorkflowMetrics tracks detailed execution metrics
type WorkflowMetrics struct {
	StepsExecuted      atomic.Int64
	StepsRetried       atomic.Int64
	StepsFailed        atomic.Int64
	WorkflowsCompleted atomic.Int64
	WorkflowsFailed    atomic.Int64
	TotalDuration      atomic.Int64 // in milliseconds
	mu                 sync.RWMutex
	StepDurations      map[string][]time.Duration
}

func NewWorkflowMetrics() *WorkflowMetrics {
	return &WorkflowMetrics{
		StepDurations: make(map[string][]time.Duration),
	}
}

func (m *WorkflowMetrics) RecordStepDuration(stepName string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StepDurations[stepName] = append(m.StepDurations[stepName], duration)
}

func (w *ComplexBillingWorkflow) Name() string {
	return w.name
}

func (w *ComplexBillingWorkflow) Run(ctx WorkflowContext, req any) error {
	startTime := time.Now()
	defer func() {
		w.metrics.TotalDuration.Add(time.Since(startTime).Milliseconds())
	}()

	// Step 1: Validate customer data
	customerID, err := Step(ctx, "validate-customer", func(stepCtx context.Context) (string, error) {
		w.metrics.StepsExecuted.Add(1)

		if w.shouldFail("validate-customer") {
			w.metrics.StepsFailed.Add(1)
			return "", fmt.Errorf("customer validation failed")
		}

		// Simulate API call
		time.Sleep(time.Duration(rand.Intn(50)+10) * time.Millisecond)
		return "customer-123", nil
	})

	if err != nil {
		// Retry with exponential backoff
		w.metrics.StepsRetried.Add(1)
		time.Sleep(100 * time.Millisecond)

		customerID, err = Step(ctx, "validate-customer-retry", func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)
			time.Sleep(time.Duration(rand.Intn(30)+20) * time.Millisecond)
			return "customer-123", nil
		})

		if err != nil {
			w.metrics.WorkflowsFailed.Add(1)
			return fmt.Errorf("customer validation failed after retry: %w", err)
		}
	}

	// Step 2: Calculate invoice amount (parallel with usage fetch)
	var invoiceAmount float64

	// Use goroutines to simulate parallel step execution
	var wg sync.WaitGroup
	var calcErr, usageErr error

	wg.Add(2)

	// Calculate invoice in parallel
	go func() {
		defer wg.Done()
		var amountStr string
		amountStr, calcErr = Step(ctx, "calculate-invoice", func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)

			if w.shouldFail("calculate-invoice") {
				w.metrics.StepsFailed.Add(1)
				return "", fmt.Errorf("invoice calculation error")
			}

			// Simulate complex calculation
			time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)
			amount := float64(rand.Intn(10000)+100) / 100.0
			return fmt.Sprintf("%.2f", amount), nil
		})

		if calcErr == nil {
			fmt.Sscanf(amountStr, "%f", &invoiceAmount)
		}
		err = calcErr
	}()

	// Fetch usage data in parallel
	go func() {
		defer wg.Done()
		_, fetchErr := Step(ctx, "fetch-usage-data", func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)

			if w.shouldFail("fetch-usage-data") {
				w.metrics.StepsFailed.Add(1)
				return "", fmt.Errorf("usage data fetch failed")
			}

			// Simulate database query
			time.Sleep(time.Duration(rand.Intn(80)+30) * time.Millisecond)
			return fmt.Sprintf("usage-%d-units", rand.Intn(1000)), nil
		})

		usageErr = fetchErr
	}()

	wg.Wait()

	if calcErr != nil || usageErr != nil {
		w.metrics.WorkflowsFailed.Add(1)
		return fmt.Errorf("parallel steps failed: calc=%v, usage=%v", calcErr, usageErr)
	}

	// Step 3: Apply discounts (conditional)
	if invoiceAmount > 100 {
		discountedAmount, discountErr := Step(ctx, "apply-discounts", func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)

			if w.shouldFail("apply-discounts") {
				w.metrics.StepsFailed.Add(1)
				return "", fmt.Errorf("discount calculation failed")
			}

			// Simulate discount calculation
			time.Sleep(time.Duration(rand.Intn(40)+10) * time.Millisecond)
			discount := invoiceAmount * 0.1
			return fmt.Sprintf("%.2f", invoiceAmount-discount), nil
		})

		if discountErr == nil {
			fmt.Sscanf(discountedAmount, "%f", &invoiceAmount)
		}
	}

	// Step 4: Generate PDF invoice
	_, err = Step(ctx, "generate-pdf", func(stepCtx context.Context) (string, error) {
		w.metrics.StepsExecuted.Add(1)

		if w.shouldFail("generate-pdf") {
			w.metrics.StepsFailed.Add(1)
			return "", fmt.Errorf("PDF generation failed")
		}

		// Simulate PDF generation (slow operation)
		time.Sleep(time.Duration(rand.Intn(200)+100) * time.Millisecond)
		return fmt.Sprintf("https://invoices.example.com/%s.pdf", customerID), nil
	})

	if err != nil {
		// Non-critical failure, continue
		// PDF generation is optional
		_ = err // Intentionally ignored
	}

	// Step 5: Send invoice email
	_, err = Step(ctx, "send-email", func(stepCtx context.Context) (string, error) {
		w.metrics.StepsExecuted.Add(1)

		if w.shouldFail("send-email") {
			w.metrics.StepsFailed.Add(1)
			return "", fmt.Errorf("email sending failed")
		}

		// Simulate email API call
		time.Sleep(time.Duration(rand.Intn(60)+20) * time.Millisecond)
		return fmt.Sprintf("email-sent-to-%s", customerID), nil
	})

	if err != nil {
		// Retry email sending
		w.metrics.StepsRetried.Add(1)
		_, retryErr := Step(ctx, "send-email-retry", func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)
			time.Sleep(time.Duration(rand.Intn(40)+20) * time.Millisecond)
			return "email-sent-on-retry", nil
		})

		if retryErr != nil {
			w.metrics.WorkflowsFailed.Add(1)
			return fmt.Errorf("email sending failed after retry: %w", retryErr)
		}
	}

	// Step 6: Update billing status
	_, err = Step(ctx, "update-billing-status", func(stepCtx context.Context) (string, error) {
		w.metrics.StepsExecuted.Add(1)

		if w.shouldFail("update-billing-status") {
			w.metrics.StepsFailed.Add(1)
			return "", fmt.Errorf("status update failed")
		}

		// Simulate database update
		time.Sleep(time.Duration(rand.Intn(30)+10) * time.Millisecond)
		return "status-updated", nil
	})

	if err != nil {
		w.metrics.WorkflowsFailed.Add(1)
		return fmt.Errorf("billing status update failed: %w", err)
	}

	w.metrics.WorkflowsCompleted.Add(1)
	return nil
}

func (w *ComplexBillingWorkflow) shouldFail(stepName string) bool {
	if !w.chaosEnabled {
		return false
	}

	// Introduce targeted chaos for specific steps
	failureRates := map[string]float64{
		"validate-customer":     w.failureRate * 0.5, // Less likely to fail
		"calculate-invoice":     w.failureRate,
		"fetch-usage-data":      w.failureRate * 1.2, // More likely to fail
		"generate-pdf":          w.failureRate * 2.0, // Much more likely to fail
		"send-email":            w.failureRate * 1.5,
		"update-billing-status": w.failureRate * 0.8,
	}

	rate, ok := failureRates[stepName]
	if !ok {
		rate = w.failureRate
	}

	return rand.Float64() < rate
}

func (w *ComplexBillingWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// ComplexDataPipelineWorkflow simulates a data processing pipeline with
// conditional branching, loops, and complex error handling
type ComplexDataPipelineWorkflow struct {
	engine       *Engine
	name         string
	chaosEnabled bool
	metrics      *WorkflowMetrics
}

func (w *ComplexDataPipelineWorkflow) Name() string {
	return w.name
}

func (w *ComplexDataPipelineWorkflow) Run(ctx WorkflowContext, req any) error {
	// Step 1: Fetch data sources
	sources, err := Step(ctx, "fetch-data-sources", func(stepCtx context.Context) ([]string, error) {
		w.metrics.StepsExecuted.Add(1)

		// Simulate fetching multiple data sources
		time.Sleep(time.Duration(rand.Intn(50)+20) * time.Millisecond)

		numSources := rand.Intn(5) + 3
		sources := make([]string, numSources)
		for i := 0; i < numSources; i++ {
			sources[i] = fmt.Sprintf("source-%d", i)
		}
		return sources, nil
	})

	if err != nil {
		w.metrics.WorkflowsFailed.Add(1)
		return fmt.Errorf("failed to fetch data sources: %w", err)
	}

	// Step 2: Process each source (loop with error handling)
	var processedCount int
	for i, source := range sources {
		stepName := fmt.Sprintf("process-source-%d", i)

		_, stepErr := Step(ctx, stepName, func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)

			// Simulate processing with variable duration
			processingTime := time.Duration(rand.Intn(100)+50) * time.Millisecond
			time.Sleep(processingTime)

			// Random failures
			if w.chaosEnabled && rand.Float64() < 0.2 {
				w.metrics.StepsFailed.Add(1)
				return "", fmt.Errorf("processing failed for %s", source)
			}

			return fmt.Sprintf("processed-%s", source), nil
		})

		if stepErr != nil {
			// Continue processing other sources
			continue
		}
		processedCount++
	}

	// Step 3: Validate processing results
	if processedCount < len(sources)/2 {
		// Too many failures, trigger cleanup
		_, cleanupErr := Step(ctx, "cleanup-failed-processing", func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)
			time.Sleep(50 * time.Millisecond)
			return "cleanup-complete", nil
		})

		if cleanupErr != nil {
			w.metrics.WorkflowsFailed.Add(1)
			return fmt.Errorf("cleanup failed: %w", cleanupErr)
		}

		w.metrics.WorkflowsFailed.Add(1)
		return fmt.Errorf("too many source processing failures: %d/%d", processedCount, len(sources))
	}

	// Step 4: Aggregate results
	_, err = Step(ctx, "aggregate-results", func(stepCtx context.Context) (string, error) {
		w.metrics.StepsExecuted.Add(1)

		// Simulate complex aggregation
		time.Sleep(time.Duration(rand.Intn(150)+100) * time.Millisecond)

		if w.chaosEnabled && rand.Float64() < 0.1 {
			w.metrics.StepsFailed.Add(1)
			return "", fmt.Errorf("aggregation failed")
		}

		return fmt.Sprintf("aggregated-%d-results", processedCount), nil
	})

	if err != nil {
		w.metrics.WorkflowsFailed.Add(1)
		return fmt.Errorf("result aggregation failed: %w", err)
	}

	// Step 5: Publish results (with circuit breaker pattern)
	var publishAttempts int
	for publishAttempts < 3 {
		_, err = Step(ctx, fmt.Sprintf("publish-attempt-%d", publishAttempts), func(stepCtx context.Context) (string, error) {
			w.metrics.StepsExecuted.Add(1)
			publishAttempts++

			// Simulate flaky external service
			if w.chaosEnabled && rand.Float64() < 0.4 {
				w.metrics.StepsFailed.Add(1)
				return "", fmt.Errorf("publish service unavailable")
			}

			time.Sleep(time.Duration(rand.Intn(80)+40) * time.Millisecond)
			return "published-successfully", nil
		})

		if err == nil {
			break
		}

		// Exponential backoff
		w.metrics.StepsRetried.Add(1)
		time.Sleep(time.Duration(publishAttempts*100) * time.Millisecond)
	}

	if err != nil {
		w.metrics.WorkflowsFailed.Add(1)
		return fmt.Errorf("failed to publish after %d attempts: %w", publishAttempts, err)
	}

	w.metrics.WorkflowsCompleted.Add(1)
	return nil
}

func (w *ComplexDataPipelineWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// ComplexStateMachineWorkflow tests complex state transitions and decision points
type ComplexStateMachineWorkflow struct {
	engine       *Engine
	name         string
	chaosEnabled bool
	metrics      *WorkflowMetrics
}

func (w *ComplexStateMachineWorkflow) Name() string {
	return w.name
}

func (w *ComplexStateMachineWorkflow) Run(ctx WorkflowContext, req any) error {
	// Initialize with random state
	initialState := rand.Intn(3)

	// Step 1: Determine initial action based on state
	action, err := Step(ctx, "determine-initial-action", func(stepCtx context.Context) (string, error) {
		w.metrics.StepsExecuted.Add(1)

		actions := []string{"process", "review", "escalate"}
		return actions[initialState], nil
	})

	if err != nil {
		w.metrics.WorkflowsFailed.Add(1)
		return err
	}

	// Step 2: Execute state machine transitions
	currentState := action
	transitions := 0
	maxTransitions := 10

	for transitions < maxTransitions {
		nextState, transitionErr := Step(ctx, fmt.Sprintf("transition-%d-from-%s", transitions, currentState),
			func(stepCtx context.Context) (string, error) {
				w.metrics.StepsExecuted.Add(1)

				// Simulate state transition logic
				time.Sleep(time.Duration(rand.Intn(50)+20) * time.Millisecond)

				// Random transition failures
				if w.chaosEnabled && rand.Float64() < 0.15 {
					w.metrics.StepsFailed.Add(1)
					return "", fmt.Errorf("transition failed from %s", currentState)
				}

				// State transition rules
				switch currentState {
				case "process":
					if rand.Float64() < 0.7 {
						return "review", nil
					}
					return "escalate", nil
				case "review":
					if rand.Float64() < 0.5 {
						return "approve", nil
					} else if rand.Float64() < 0.8 {
						return "reject", nil
					}
					return "process", nil
				case "escalate":
					if rand.Float64() < 0.6 {
						return "review", nil
					}
					return "terminate", nil
				case "approve", "reject", "terminate":
					return currentState, nil // Terminal states
				default:
					return "error", nil
				}
			})

		if transitionErr != nil {
			// Handle transition failure
			_, recoveryErr := Step(ctx, fmt.Sprintf("recover-transition-%d", transitions),
				func(stepCtx context.Context) (string, error) {
					w.metrics.StepsExecuted.Add(1)
					w.metrics.StepsRetried.Add(1)
					time.Sleep(30 * time.Millisecond)
					return "review", nil // Safe state
				})

			if recoveryErr != nil {
				w.metrics.WorkflowsFailed.Add(1)
				return fmt.Errorf("state machine recovery failed: %w", recoveryErr)
			}
			nextState = "review"
		}

		currentState = nextState
		transitions++

		// Check for terminal states
		if currentState == "approve" || currentState == "reject" || currentState == "terminate" {
			break
		}
	}

	// Step 3: Finalize based on terminal state
	_, err = Step(ctx, fmt.Sprintf("finalize-%s", currentState), func(stepCtx context.Context) (string, error) {
		w.metrics.StepsExecuted.Add(1)

		switch currentState {
		case "approve":
			time.Sleep(80 * time.Millisecond)
			return "approved-and-processed", nil
		case "reject":
			time.Sleep(40 * time.Millisecond)
			return "rejected-and-notified", nil
		case "terminate":
			time.Sleep(20 * time.Millisecond)
			return "terminated-with-cleanup", nil
		default:
			return "", fmt.Errorf("invalid terminal state: %s", currentState)
		}
	})

	if err != nil {
		w.metrics.WorkflowsFailed.Add(1)
		return fmt.Errorf("finalization failed: %w", err)
	}

	w.metrics.WorkflowsCompleted.Add(1)
	return nil
}

func (w *ComplexStateMachineWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
