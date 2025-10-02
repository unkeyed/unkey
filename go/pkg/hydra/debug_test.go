package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
)

// TestBasicWorkflowExecution tests the most basic workflow execution
func TestBasicWorkflowExecution(t *testing.T) {
	realClock := clock.New()
	engine := newTestEngineWithClock(t, realClock)

	// Create a very simple workflow
	simpleWorkflow := &debugWorkflow{
		engine: engine,
		name:   "debug-workflow",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start worker
	worker, err := NewWorker(engine, WorkerConfig{
		WorkerID:          "debug-worker",
		Concurrency:       1,
		PollInterval:      100 * time.Millisecond,
		HeartbeatInterval: 2 * time.Second,
		ClaimTimeout:      10 * time.Second,
	})
	require.NoError(t, err)

	err = RegisterWorkflow(worker, simpleWorkflow)
	require.NoError(t, err)

	err = worker.Start(ctx)
	require.NoError(t, err)
	defer worker.Shutdown(ctx)

	// Submit a single workflow
	workflowID, err := simpleWorkflow.Start(ctx, "test-payload")
	require.NoError(t, err)
	require.NotEmpty(t, workflowID)

	// Wait for completion
	finalWorkflow := waitForWorkflowCompletion(t, engine, workflowID, 8*time.Second)
	require.NotNil(t, finalWorkflow, "Workflow should complete")

}

type debugWorkflow struct {
	engine *Engine
	name   string
}

func (w *debugWorkflow) Name() string {
	return w.name
}

func (w *debugWorkflow) Run(ctx WorkflowContext, req any) error {
	// Very simple step
	_, err := Step(ctx, "debug-step", func(context.Context) (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "success", nil
	})
	return err
}

func (w *debugWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}
