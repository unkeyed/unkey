package hydra

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/hydra/store/gorm"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"gorm.io/driver/mysql"
	gormlib "gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Test helper functions

// newTestEngine creates an engine instance for testing with MySQL
func newTestEngine(t *testing.T) *Engine {
	return newTestEngineWithClock(t, clock.New())
}

// newTestEngineWithClock creates an engine instance with a controllable test clock and MySQL
func newTestEngineWithClock(t *testing.T, clk clock.Clock) *Engine {
	t.Helper()

	// Use MySQL container for testing
	containersClient := containers.New(t)
	hostDsn, _ := containersClient.RunMySQL()

	// Open MySQL database with GORM
	db, err := gormlib.Open(mysql.Open(hostDsn), &gormlib.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)

	// Auto-migrate the hydra schema
	err = db.AutoMigrate(
		&store.WorkflowExecution{},
		&store.WorkflowStep{},
		&store.CronJob{},
		&store.Lease{},
	)
	require.NoError(t, err)

	// Create the store
	gormStore := gorm.NewGORMStore(db, clk)

	// Create engine with unique namespace for test isolation
	testNamespace := uid.New(uid.TestPrefix)
	
	engine := New(Config{
		Store:     gormStore,
		Namespace: testNamespace,
		Clock:     clk,
	})

	return engine
}

// waitForWorkflowCompletion waits until a workflow reaches a terminal state (completed or failed)
func waitForWorkflowCompletion(t *testing.T, e *Engine, executionID string, timeout time.Duration) *store.WorkflowExecution {
	t.Helper()

	var workflow *store.WorkflowExecution

	require.Eventually(t, func() bool {
		var err error
		workflow, err = e.GetStore().GetWorkflow(context.Background(), e.GetNamespace(), executionID)
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
