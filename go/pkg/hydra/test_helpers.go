package hydra

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// newTestEngineWithClock creates a test engine with the specified clock
func newTestEngineWithClock(t *testing.T, clk clock.Clock) *Engine {
	t.Helper()

	// Use testcontainers for MySQL
	mysqlCfg := containers.MySQL(t)
	mysqlCfg.DBName = "hydra"
	hydraDsn := mysqlCfg.FormatDSN()

	// Load the hydra schema into the database
	db, err := sql.Open("mysql", hydraDsn)
	require.NoError(t, err)
	defer db.Close()

	// Create a unique namespace for this test to avoid data pollution
	testNamespace := fmt.Sprintf("test_%s_%s", t.Name(), uid.New(uid.Prefix("test")))

	// Create the engine with the properly configured database
	engine, err := New(Config{
		DSN:        hydraDsn,
		Namespace:  testNamespace,
		Clock:      clk,
		Logger:     logging.NewNoop(),
		Marshaller: NewJSONMarshaller(),
	})
	if err != nil {
		t.Fatalf("Failed to create test engine: %v", err)
	}

	return engine
}

// newTestEngine creates a test engine with default clock
func newTestEngine(t *testing.T) *Engine {
	return newTestEngineWithClock(t, clock.New())
}

// waitForWorkflowCompletion waits for a workflow to complete and returns the final workflow state
func waitForWorkflowCompletion(t *testing.T, engine *Engine, workflowID string, timeout time.Duration) *store.WorkflowExecution {
	t.Helper()

	var workflow store.WorkflowExecution
	var err error

	require.Eventually(t, func() bool {
		workflow, err = store.Query.GetWorkflow(context.Background(), engine.GetDB(), store.GetWorkflowParams{
			ID:        workflowID,
			Namespace: engine.GetNamespace(),
		})
		if err != nil {
			return false
		}
		return workflow.Status == store.WorkflowExecutionsStatusCompleted ||
			workflow.Status == store.WorkflowExecutionsStatusFailed
	}, timeout, 100*time.Millisecond, "Workflow should complete within timeout")

	require.NoError(t, err)
	return &workflow
}
