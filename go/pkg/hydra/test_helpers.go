package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
)

// Test helper functions

// newTestEngine creates an engine instance for testing with temp file SQLite
func newTestEngine(t *testing.T) *Engine {
	// Create unique temp directory for this test
	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"

	store, err := gorm.NewSQLiteStore(dbPath, nil)
	require.NoError(t, err)

	return New(Config{
		Store: store,
		Clock: clock.New(), // Use real clock for non-clock tests
	})
}

// newTestEngineWithClock creates an engine instance with a controllable test clock
func newTestEngineWithClock(t *testing.T, clk clock.Clock) *Engine {
	// Create unique temp directory for this test
	tempDir := t.TempDir()
	dbPath := tempDir + "/test.db"

	store, err := gorm.NewSQLiteStore(dbPath, clk)
	require.NoError(t, err)

	return New(Config{
		Store: store,
		Clock: clk,
	})
}

// waitForWorkflowCompletion waits until a workflow reaches a terminal state (completed or failed)
func waitForWorkflowCompletion(t *testing.T, e *Engine, executionID string, timeout time.Duration) *store.WorkflowExecution {
	t.Helper()

	var workflow *store.WorkflowExecution

	require.Eventually(t, func() bool {
		var err error
		workflow, err = e.GetStore().GetWorkflow(context.Background(), "default", executionID)
		if err != nil {
			t.Logf("Error getting workflow %s: %v", executionID, err)
			return false
		}

		isComplete := workflow.Status == store.WorkflowStatusCompleted ||
			workflow.Status == store.WorkflowStatusFailed

		if !isComplete {
			t.Logf("Workflow %s status: %s", executionID, workflow.Status)
		}

		return isComplete
	}, timeout, 100*time.Millisecond, "Workflow should complete within timeout")

	return workflow
}

// waitForWorkflowSuccess waits until a workflow completes successfully
func waitForWorkflowSuccess(t *testing.T, e *Engine, executionID string, timeout time.Duration) *store.WorkflowExecution {
	t.Helper()

	workflow := waitForWorkflowCompletion(t, e, executionID, timeout)
	require.Equal(t, store.WorkflowStatusCompleted, workflow.Status, "Workflow should complete successfully")

	return workflow
}

// waitForWorkflowFailure waits until a workflow fails
func waitForWorkflowFailure(t *testing.T, e *Engine, executionID string, timeout time.Duration) *store.WorkflowExecution {
	t.Helper()

	workflow := waitForWorkflowCompletion(t, e, executionID, timeout)
	require.Equal(t, store.WorkflowStatusFailed, workflow.Status, "Workflow should fail")

	return workflow
}
