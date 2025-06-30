package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
	"github.com/stretchr/testify/require"
)

// Test helper functions

// newTestEngine creates an engine instance for testing with in-memory SQLite
func newTestEngine(t *testing.T) *Engine {
	store, err := gorm.NewSQLiteStore(":memory:", nil)
	require.NoError(t, err)
	
	return New(Config{Store: store})
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