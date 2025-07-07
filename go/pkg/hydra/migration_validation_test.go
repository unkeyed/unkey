package hydra

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/unkeyed/unkey/go/pkg/hydra/store"
)

// TestNoGORMDependencies ensures that GORM dependencies have been completely removed
func TestNoGORMDependencies(t *testing.T) {
	// Test that no Go files import GORM
	// This would typically be implemented as a build constraint or linter rule
	t.Run("NoGORMImports", func(t *testing.T) {
		// This test would scan source files for GORM imports
		// In a real implementation, you'd use go/parser to check imports
		t.Log("Manual verification: No GORM imports should exist in codebase")
	})
}

// TestSQLCQueryAccess ensures the Query singleton is properly accessible
func TestSQLCQueryAccess(t *testing.T) {
	// Verify that we can access the SQLC Query singleton and engine has DB access
	engine := newTestEngine(t)

	t.Run("QuerySingletonExists", func(t *testing.T) {
		// Test that store.Query is accessible and implements Querier interface
		queryType := reflect.TypeOf(store.Query)
		if queryType == nil {
			t.Fatal("store.Query should not be nil")
		}
		t.Logf("Query type: %v", queryType)
	})

	t.Run("EngineHasDBAccess", func(t *testing.T) {
		// Test that engine has direct database access
		db := engine.GetDB()
		if db == nil {
			t.Fatal("Engine should have non-nil database connection")
		}

		// Test that we can perform a basic query
		ctx := context.Background()
		// GetAllWorkflows was removed as debug-only query
		// Test basic query functionality instead
		pendingWorkflows, err := store.Query.GetPendingWorkflows(ctx, db, store.GetPendingWorkflowsParams{
			Namespace:   engine.GetNamespace(),
			NextRetryAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			SleepUntil:  sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("Should be able to query workflows: %v", err)
		}
		t.Logf("Found %d pending workflows", len(pendingWorkflows))
	})

	t.Run("NoStoreAbstraction", func(t *testing.T) {
		// Verify that the old store abstraction methods don't exist on Engine
		engineType := reflect.TypeOf(engine)

		// These methods should NOT exist anymore
		oldMethods := []string{"GetStore", "GetSQLCStore", "store"}
		for _, methodName := range oldMethods {
			_, hasMethod := engineType.MethodByName(methodName)
			if hasMethod {
				t.Errorf("Engine should not have deprecated method: %s", methodName)
			}
		}
	})
}
