//go:build integration

package multi_node_usagelimiting

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// Constants for database consistency polling
const (
	defaultReplayTimeout      = 10 * time.Second
	defaultReplayPollInterval = 200 * time.Millisecond
)

// Constants for accuracy tolerance thresholds
const (
	// lowLoadAccuracyTolerance is the tolerance for low load scenarios (0% - perfect accuracy expected)
	lowLoadAccuracyTolerance = 0.0
	
	// highLoadAccuracyTolerance is the maximum tolerance for high contention scenarios (1% maximum)
	highLoadAccuracyTolerance = 0.01
)

// waitForDatabaseConsistency polls the database until the remaining credits match exactly
// the expected value or the timeout is reached using require.Eventually.
func waitForDatabaseConsistency(t *testing.T, ctx context.Context, dbConn db.DBTX, 
	keyID string, expectedRemaining int64, timeout time.Duration, pollInterval time.Duration) (int32, error) {
	
	var finalRemaining int32
	var lastErr error
	
	require.Eventually(t, func() bool {
		finalKey, err := db.Query.FindKeyByID(ctx, dbConn, keyID)
		if err != nil {
			lastErr = err
			return false
		}
		
		if finalKey.RemainingRequests.Valid {
			finalRemaining = finalKey.RemainingRequests.Int32
			return int64(finalRemaining) == expectedRemaining
		}
		
		return false
	}, timeout, pollInterval, "Database should reach exact consistency within timeout")
	
	return finalRemaining, lastErr
}

// TestUsageLimitAccuracy tests the accuracy of credit counting under high concurrency
func TestUsageLimitAccuracy(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	testCases := []struct {
		name         string
		nodeCount    int
		totalCredits int64
		cost         int64
		concurrency  int
		requests     int
	}{
		{
			name:         "SingleNode_HighConcurrency",
			nodeCount:    1,
			totalCredits: 100,
			cost:         1,
			concurrency:  50,
			requests:     200,
		},
		{
			name:         "MultiNode_HighConcurrency",
			nodeCount:    3,
			totalCredits: 100,
			cost:         1,
			concurrency:  100,
			requests:     300,
		},
		{
			name:         "MultiNode_HighCost",
			nodeCount:    5,
			totalCredits: 500,
			cost:         10,
			concurrency:  20,
			requests:     100,
		},
		{
			name:         "ExtremeLoad_SingleCredit",
			nodeCount:    3,
			totalCredits: 1,
			cost:         1,
			concurrency:  1000,
			requests:     1000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runAccuracyTest(t, tc.nodeCount, tc.totalCredits, tc.cost, tc.concurrency, tc.requests)
		})
	}
}

func runAccuracyTest(t *testing.T, nodeCount int, totalCredits, cost int64, concurrency, totalRequests int) {
	ctx := context.Background()

	h := integration.New(t, integration.Config{
		NumNodes: nodeCount,
	})

	// Step 1: Set up test resources
	workspace := h.Resources().UserWorkspace
	rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

	api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:   api.KeyAuthID.String,
		Remaining:   ptr.P(int32(totalCredits)),
	})

	keyStart := keyResponse.Key

	// Step 2: Run concurrent load test
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Key: keyStart,
		Credits: &openapi.KeysVerifyKeyCredits{
			Cost: int32(cost),
		},
	}

	lb := integration.NewLoadbalancer(h)

	// Track results
	var mu sync.Mutex
	successCount := 0
	failureCount := 0
	var minRemainingCredits int64 = totalCredits // Track the minimum remaining credits seen

	// Create worker goroutines
	var wg sync.WaitGroup
	requestChan := make(chan int, totalRequests)

	// Fill request channel
	for i := 0; i < totalRequests; i++ {
		requestChan <- i
	}
	close(requestChan)

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for reqNum := range requestChan {
				// Add some jitter to simulate real-world timing
				needSleep := (reqNum%10 == 0)
				jitter := time.Millisecond * time.Duration(workerID%5)
				
				if needSleep {
					time.Sleep(jitter)
				}

				res, callErr := integration.CallRandomNode[handler.Request, handler.Response](
					lb, "POST", "/v2/keys.verifyKey", headers, req)

				require.NoError(t, callErr)
				if res.Status != 200 {
					t.Logf("Request failed with status %d. Response body: %+v", res.Status, res.Body)
				}
				require.Equal(t, 200, res.Status)

				mu.Lock()
				if res.Body.Data.Valid {
					successCount++
					if res.Body.Data.Credits != nil {
						currentRemaining := int64(*res.Body.Data.Credits)
						if currentRemaining < minRemainingCredits {
							minRemainingCredits = currentRemaining
						}
					}
				} else {
					failureCount++
				}
				mu.Unlock()
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Step 3: Verify accuracy
	expectedSuccessful := int(totalCredits / cost)
	totalProcessed := successCount + failureCount

	t.Logf("Results: %d successful, %d failed, %d total", successCount, failureCount, totalProcessed)
	t.Logf("Expected successful: %d, Actual successful: %d", expectedSuccessful, successCount)
	t.Logf("Minimum remaining credits seen: %d", minRemainingCredits)

	// Verify total requests processed
	require.Equal(t, totalRequests, totalProcessed, "All requests should be processed")

	// Verify credit accuracy - should not exceed the limit
	require.LessOrEqual(t, successCount, expectedSuccessful, "Should not exceed credit limit")

	// With atomic Redis operations, we expect perfect accuracy on low load
	// Only allow error margin on high contention scenarios
	if totalRequests <= expectedSuccessful {
		// Low load - expect perfect accuracy (0% tolerance)
		require.Equal(t, min(totalRequests, expectedSuccessful), successCount,
			"Low load should have perfect accuracy with atomic operations")
	} else {
		// High contention - allow maximum 1% error margin for race conditions
		minExpected := max(0, int(float64(expectedSuccessful)*(1.0-highLoadAccuracyTolerance)))
		require.GreaterOrEqual(t, successCount, minExpected, 
			"Should not under-count by more than %.0f%% on high load", highLoadAccuracyTolerance*100)
	}

	// Verify remaining credits accuracy - should be 100% accurate
	expectedRemaining := totalCredits - int64(successCount)*cost
	require.Equal(t, expectedRemaining, minRemainingCredits,
		"Remaining credits must be 100%% accurate")

	// Verify all nodes received traffic in multi-node tests
	if nodeCount > 1 {
		lbMetrics := lb.GetMetrics()
		require.Equal(t, nodeCount, len(lbMetrics), "All nodes should have received traffic")

		// Verify traffic distribution is reasonable (no node should handle more than 70% of traffic)
		for nodeID, count := range lbMetrics {
			percentage := float64(count) / float64(totalRequests) * 100
			require.LessOrEqual(t, percentage, 70.0,
				"Node %s handled %.1f%% of traffic (should be <= 70%%)", nodeID, percentage)
		}
	}

	// Step 4: Verify final state in database with polling-based wait
	t.Logf("Waiting for database consistency (timeout: %v, poll interval: %v)", 
		defaultReplayTimeout, defaultReplayPollInterval)
	
	dbRemaining, err := waitForDatabaseConsistency(t, ctx, h.DB.RO(), keyResponse.KeyID, 
		expectedRemaining, defaultReplayTimeout, defaultReplayPollInterval)
	require.NoError(t, err, "Database query should not fail during consistency check")
	
	t.Logf("Database remaining credits: %d", dbRemaining)
	
	// Verify the final consistency - must be 100% accurate
	require.Equal(t, expectedRemaining, int64(dbRemaining),
		"Database remaining credits must be 100%% accurate after replay")
}
