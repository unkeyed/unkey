package hydra

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/db"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
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

	// Open MySQL database with sql.DB
	sqlDB, err := sql.Open("mysql", hostDsn)
	require.NoError(t, err)

	// Test the connection
	err = sqlDB.Ping()
	require.NoError(t, err)

	// TODO: Run schema migrations for sqlc tables
	// This would typically use the schema.sql file from the db package
	// For now, we'll assume the schema is already set up

	// Create engine with unique namespace for test isolation
	testNamespace := uid.New(uid.TestPrefix)

	engine := New(Config{
		DB:         sqlDB,
		Namespace:  testNamespace,
		Clock:      clk,
		Logger:     nil,
		Marshaller: nil,
	})

	return engine
}

// waitForWorkflowCompletion waits until a workflow reaches a terminal state (completed or failed)
func waitForWorkflowCompletion(t *testing.T, e *Engine, executionID string, timeout time.Duration) *db.WorkflowExecution {
	t.Helper()

	var workflow *db.WorkflowExecution

	require.Eventually(t, func() bool {
		var err error
		wf, err := db.Query.GetWorkflow(context.Background(), e.db, db.GetWorkflowParams{
			ID:        executionID,
			Namespace: e.GetNamespace(),
		})
		if err != nil {
			t.Logf("Error getting workflow %s: %v", executionID, err)
			return false
		}
		workflow = &wf

		isComplete := workflow.Status == db.WorkflowExecutionsStatusCompleted ||
			workflow.Status == db.WorkflowExecutionsStatusFailed

		if !isComplete {
			t.Logf("Workflow %s status: %s", executionID, workflow.Status)
		}

		return isComplete
	}, timeout, 100*time.Millisecond, "Workflow should complete within timeout")

	return workflow
}
