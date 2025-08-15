//go:build integration

package multi_node_usagelimiting

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

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
			concurrency:  100,
			requests:     100,
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
		KeyAuthID:   api.KeyAuthID.String,
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

	t.Logf("Test request: Key=%s, Cost=%d", keyStart, cost)

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
				if reqNum%10 == 0 {
					time.Sleep(time.Millisecond * time.Duration(workerID%5))
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
	assert.Equal(t, totalRequests, totalProcessed, "All requests should be processed")

	// Verify credit accuracy - should not exceed the limit
	assert.LessOrEqual(t, successCount, expectedSuccessful, "Should not exceed credit limit")

	// With atomic Redis operations, we expect perfect accuracy on low load
	// Only allow error margin on high contention scenarios
	if totalRequests <= expectedSuccessful {
		// Low load - expect perfect accuracy
		assert.Equal(t, min(totalRequests, expectedSuccessful), successCount, 
			"Low load should have perfect accuracy with atomic operations")
	} else {
		// High contention - allow small error margin for race conditions
		minExpected := max(0, int(float64(expectedSuccessful)*0.99))
		assert.GreaterOrEqual(t, successCount, minExpected, "Should not under-count by more than 1%% on high load")
	}

	// Verify remaining credits accuracy using the minimum observed value
	expectedRemaining := totalCredits - int64(successCount)*cost
	assert.InDelta(t, expectedRemaining, minRemainingCredits, float64(cost)*2,
		"Minimum remaining credits should be accurate within 2 cost units")

	// Verify all nodes received traffic in multi-node tests
	if nodeCount > 1 {
		lbMetrics := lb.GetMetrics()
		assert.Equal(t, nodeCount, len(lbMetrics), "All nodes should have received traffic")

		// Verify traffic distribution is reasonable (no node should handle more than 70% of traffic)
		for nodeID, count := range lbMetrics {
			percentage := float64(count) / float64(totalRequests) * 100
			assert.LessOrEqual(t, percentage, 70.0,
				"Node %s handled %.1f%% of traffic (should be <= 70%%)", nodeID, percentage)
		}
	}

	// Step 4: Verify final state in database
	time.Sleep(2 * time.Second) // Allow time for async replay to complete

	finalKey, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)

	if finalKey.RemainingRequests.Valid {
		dbRemaining := finalKey.RemainingRequests.Int32
		t.Logf("Database remaining credits: %d", dbRemaining)

		// DB should be eventually consistent with Redis
		assert.InDelta(t, expectedRemaining, dbRemaining, float64(cost)*5,
			"Database remaining credits should be eventually consistent")
	}
}
