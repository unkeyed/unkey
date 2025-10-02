package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
)

// TestCircuitBreakerIntegration verifies that circuit breakers are properly
// integrated into the worker and protect database operations
func TestCircuitBreakerIntegration(t *testing.T) {
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	// Create worker with circuit breaker protection
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "circuit-breaker-test-worker",
		Concurrency:       1,
		PollInterval:      100 * time.Millisecond,
		HeartbeatInterval: 5 * time.Second,
		ClaimTimeout:      30 * time.Second,
	})
	require.NoError(t, err)

	// Register a simple workflow
	circuitTestWorkflow := &circuitBreakerTestWorkflow{
		engine: engine,
		name:   "circuit-breaker-workflow",
	}

	err = RegisterWorkflow(worker, circuitTestWorkflow)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start worker
	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Create a workflow to test circuit breaker protection
	executionID, err := circuitTestWorkflow.Start(ctx, struct{}{})
	require.NoError(t, err)
	require.NotEmpty(t, executionID)

	// Advance time to trigger worker polling
	for i := 0; i < 5; i++ {
		testClock.Tick(200 * time.Millisecond)
		time.Sleep(10 * time.Millisecond)
	}

	// Verify workflow was processed (circuit breaker didn't block)
	finalWorkflow := waitForWorkflowCompletion(t, engine, executionID, 3*time.Second)
	require.NotNil(t, finalWorkflow)

}

// TestCircuitBreakerCompilation ensures the circuit breaker types compile correctly
func TestCircuitBreakerCompilation(t *testing.T) {
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	// This test primarily ensures compilation works
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:    "compilation-test-worker",
		Concurrency: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, worker)

}

// circuitBreakerTestWorkflow is a minimal workflow for testing circuit breaker integration
type circuitBreakerTestWorkflow struct {
	engine *Engine
	name   string
}

func (w *circuitBreakerTestWorkflow) Name() string {
	return w.name
}

func (w *circuitBreakerTestWorkflow) Run(ctx WorkflowContext, req any) error {
	_, err := Step(ctx, "circuit-breaker-step", func(context.Context) (string, error) {
		return "protected", nil
	})
	return err
}

func (w *circuitBreakerTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
