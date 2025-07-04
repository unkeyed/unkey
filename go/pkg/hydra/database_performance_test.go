package hydra

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/hydra/store"
)

// loadTestWorkflow is a minimal workflow for load testing
type loadTestWorkflow struct {
	engine *Engine
	name   string
}

func (w *loadTestWorkflow) Name() string {
	return w.name
}

func (w *loadTestWorkflow) Run(ctx WorkflowContext, req any) error {
	// Minimal work to avoid affecting performance measurements
	_, err := Step(ctx, "load-test-step", func(context.Context) (string, error) {
		return "done", nil
	})
	return err
}

func (w *loadTestWorkflow) Start(ctx context.Context, payload any) (string, error) {
	return w.engine.StartWorkflow(ctx, w.Name(), payload)
}

// TestDatabaseQueryPerformanceUnderLoad tests how database queries perform
// when the system has thousands of pending workflows and multiple workers
// are polling concurrently.
func TestDatabaseQueryPerformanceUnderLoad(t *testing.T) {
	// Arrange: Create engine with many pending workflows
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)

	const (
		numPendingWorkflows  = 1000 // Create backlog to stress query performance
		numConcurrentWorkers = 10   // Multiple workers polling simultaneously
		testDurationSec      = 5    // Run load test for 5 seconds
	)

	// Create a large backlog of pending workflows to stress GetPendingWorkflows query
	executionIDs := make([]string, numPendingWorkflows)

	loadTestWorkflow := &loadTestWorkflow{
		engine: engine,
		name:   "load-test-workflow",
	}

	ctx := context.Background()
	for i := 0; i < numPendingWorkflows; i++ {
		executionID, err := loadTestWorkflow.Start(ctx, struct{}{})
		require.NoError(t, err)
		executionIDs[i] = executionID
	}

	// Verify workflows are pending
	pendingCount, err := countPendingWorkflows(engine.GetSQLCStore(), ctx, engine.GetNamespace())
	require.NoError(t, err)
	require.Equal(t, numPendingWorkflows, pendingCount,
		"Should have created %d pending workflows", numPendingWorkflows)

	// Performance tracking
	var totalQueries atomic.Int64
	var totalQueryTime atomic.Int64 // nanoseconds
	var slowQueries atomic.Int64    // queries > 100ms
	var errors atomic.Int64

	// Act: Start multiple workers that will poll the database concurrently
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(testDurationSec)*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Start concurrent workers that stress GetPendingWorkflows query
	for i := 0; i < numConcurrentWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Measure query performance
					start := time.Now()
					_, err := engine.GetSQLCStore().GetPendingWorkflows(ctx, engine.GetNamespace(), 50, nil)
					queryDuration := time.Since(start)

					totalQueries.Add(1)
					totalQueryTime.Add(int64(queryDuration))

					if err != nil {
						errors.Add(1)
						// Don't spam error logs during load test
						if errors.Load() <= 10 {
							t.Logf("Error #%d: %v", errors.Load(), err)
						}
					}

					// Track slow queries (potential performance issue)
					if queryDuration > 100*time.Millisecond {
						slowQueries.Add(1)
						if slowQueries.Load() <= 10 {
							t.Logf("Slow query #%d: %v", slowQueries.Load(), queryDuration)
						}
					}

					// Small pause to prevent overwhelming the database
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()
	}

	// Let the load test run

	wg.Wait()

	// Assert: Analyze performance results
	finalQueries := totalQueries.Load()
	finalQueryTime := totalQueryTime.Load()
	finalSlowQueries := slowQueries.Load()
	finalErrors := errors.Load()

	avgQueryTime := time.Duration(finalQueryTime / finalQueries)
	queriesPerSecond := float64(finalQueries) / float64(testDurationSec)
	slowQueryPercentage := float64(finalSlowQueries) / float64(finalQueries) * 100
	errorPercentage := float64(finalErrors) / float64(finalQueries) * 100

	// Performance assertions
	require.Less(t, avgQueryTime, 100*time.Millisecond,
		"Average query time should be <100ms even with %d pending workflows", numPendingWorkflows)

	require.Less(t, slowQueryPercentage, 10.0,
		"Less than 10%% of queries should be slow (>100ms), got %.1f%%", slowQueryPercentage)

	require.Less(t, errorPercentage, 5.0,
		"Less than 5%% of queries should error, got %.1f%%", errorPercentage)

	require.Greater(t, queriesPerSecond, 50.0,
		"Should sustain >50 queries/sec under load, got %.1f", queriesPerSecond)

}

// TestDatabaseIndexOptimization verifies that database queries use appropriate indexes
// and don't degrade significantly as the number of workflows grows
func TestDatabaseIndexOptimization(t *testing.T) {
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)
	ctx := context.Background()

	// Test query performance with different workflow counts
	testCases := []struct {
		workflowCount int
		maxQueryTime  time.Duration
	}{
		{100, 10 * time.Millisecond},
		{1000, 50 * time.Millisecond},
		{5000, 100 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("workflows_%d", tc.workflowCount), func(t *testing.T) {
			// Clean slate for each test
			engine = newTestEngineWithClock(t, testClock)

			// Create new workflow instance for this engine
			testWorkflow := &loadTestWorkflow{
				engine: engine,
				name:   "index-test-workflow",
			}

			// Create workflows
			for i := 0; i < tc.workflowCount; i++ {
				_, err := testWorkflow.Start(ctx, struct{}{})
				require.NoError(t, err)
			}

			// Measure query performance
			const numQueries = 10
			var totalTime time.Duration

			for i := 0; i < numQueries; i++ {
				start := time.Now()
				workflows, err := engine.GetSQLCStore().GetPendingWorkflows(ctx, engine.GetNamespace(), 50, nil)
				duration := time.Since(start)

				require.NoError(t, err)
				require.NotEmpty(t, workflows, "Should find pending workflows")

				totalTime += duration
			}

			avgTime := totalTime / numQueries

			require.Less(t, avgTime, tc.maxQueryTime,
				"Query time should scale well with workflow count")
		})
	}
}

// TestConcurrentLeaseAcquisition tests how the database handles multiple workers
// trying to acquire leases on the same workflows simultaneously
func TestConcurrentLeaseAcquisition(t *testing.T) {
	testClock := clock.NewTestClock()
	engine := newTestEngineWithClock(t, testClock)
	ctx := context.Background()

	const numWorkflows = 100
	const numWorkers = 20

	// Create pending workflows
	loadTestWorkflow := &loadTestWorkflow{
		engine: engine,
		name:   "lease-test-workflow",
	}

	executionIDs := make([]string, numWorkflows)
	for i := 0; i < numWorkflows; i++ {
		executionID, err := loadTestWorkflow.Start(ctx, struct{}{})
		require.NoError(t, err)
		executionIDs[i] = executionID
	}

	// Track lease acquisition results
	var successfulLeases atomic.Int64
	var failedLeases atomic.Int64
	var duplicateLeases atomic.Int64

	leaseOwnership := make(map[string]string) // workflowID -> workerID
	var mu sync.Mutex

	// Start workers trying to acquire leases concurrently
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerIDStr := fmt.Sprintf("worker-%d", workerID)

			for _, executionID := range executionIDs {
				// Try to acquire lease
				lease := &store.Lease{
					ResourceID:  executionID,
					Kind:        "workflow",
					Namespace:   engine.GetNamespace(),
					WorkerID:    workerIDStr,
					AcquiredAt:  testClock.Now().UnixMilli(),
					ExpiresAt:   testClock.Now().Add(30 * time.Second).UnixMilli(),
					HeartbeatAt: testClock.Now().UnixMilli(),
				}
				err := engine.store.AcquireLease(ctx, lease)

				if err != nil {
					failedLeases.Add(1)
					continue
				}

				// Check for duplicate lease (data race/corruption)
				mu.Lock()
				if existingWorker, exists := leaseOwnership[executionID]; exists {
					duplicateLeases.Add(1)
					t.Errorf("DUPLICATE LEASE: Workflow %s acquired by both %s and %s",
						executionID, existingWorker, workerIDStr)
				} else {
					leaseOwnership[executionID] = workerIDStr
					successfulLeases.Add(1)
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Analyze results
	successful := successfulLeases.Load()
	_ = failedLeases.Load()
	duplicates := duplicateLeases.Load()

	// Assertions for data integrity
	require.Equal(t, int64(0), duplicates,
		"No duplicate leases should occur - indicates race condition")

	require.Equal(t, int64(numWorkflows), successful,
		"All workflows should be successfully leased exactly once")

	// Most failures are expected due to workers competing for same workflows
}

// Helper function to count pending workflows
func countPendingWorkflows(s store.Store, ctx context.Context, namespace string) (int, error) {
	workflows, err := s.GetPendingWorkflows(ctx, namespace, 10000, nil) // Large limit
	if err != nil {
		return 0, err
	}
	return len(workflows), nil
}
