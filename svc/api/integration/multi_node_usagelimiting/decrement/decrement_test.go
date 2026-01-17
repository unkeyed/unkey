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
	"github.com/unkeyed/unkey/svc/api/integration"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// TestDecrementAccuracy tests the decrement logic through the verify key endpoint
func TestDecrementAccuracy(t *testing.T) {


	testCases := []struct {
		name         string
		nodeCount    int
		totalCredits int64
		cost         int64
		concurrency  int
		requests     int
	}{
		{
			name:         "BasicDecrement_SingleNode",
			nodeCount:    1,
			totalCredits: 50,
			cost:         1,
			concurrency:  10,
			requests:     75, // 25 more than available credits
		},
		{
			name:         "HighCostDecrement_MultiNode",
			nodeCount:    3,
			totalCredits: 100,
			cost:         5,
			concurrency:  20,
			requests:     50, // Should succeed 20 times (20*5=100), fail 30 times
		},
		{
			name:         "SmallCredits_HighContention",
			nodeCount:    5,
			totalCredits: 5,
			cost:         1,
			concurrency:  50,
			requests:     50, // Only first 5 should succeed
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			h := integration.New(t, integration.Config{
				NumNodes: tc.nodeCount,
			})

			// Set up test resources
			workspace := h.Resources().UserWorkspace
			rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

			api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
				WorkspaceID: workspace.ID,
			})

			keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeySpaceID:   api.KeyAuthID.String,
				Remaining:   ptr.P(int32(tc.totalCredits)),
			})

			// Set up request
			req := handler.Request{
				Key: keyResponse.Key,
			}
			if tc.cost > 1 {
				req.Credits = &openapi.KeysVerifyKeyCredits{
					Cost: int32(tc.cost),
				}
			}

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			lb := integration.NewLoadbalancer(h)

			// Track results
			var mu sync.Mutex
			successCount := 0
			failureCount := 0
			var minRemainingCredits int64 = tc.totalCredits
			totalRequests := tc.requests

			// Create request channel
			requestChan := make(chan int, totalRequests)
			for i := 0; i < totalRequests; i++ {
				requestChan <- i
			}
			close(requestChan)

			// Start concurrent workers
			var wg sync.WaitGroup
			for i := 0; i < tc.concurrency; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for reqNum := range requestChan {
						// Add realistic jitter
						needSleep := (reqNum%10 == 0)
						jitter := time.Millisecond * time.Duration(workerID%5)

						if needSleep {
							time.Sleep(jitter)
						}

						res, callErr := integration.CallRandomNode[handler.Request, handler.Response](
							lb, "POST", "/v2/keys.verifyKey", headers, req)

						require.NoError(t, callErr)
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

			wg.Wait()

			// Calculate expected results
			expectedSuccessful := int(tc.totalCredits / tc.cost)
			expectedFailures := totalRequests - expectedSuccessful
			expectedRemaining := tc.totalCredits - int64(expectedSuccessful)*tc.cost

			t.Logf(" decrement test results:")
			t.Logf("  Total requests: %d", totalRequests)
			t.Logf("  Successful: %d (expected: %d)", successCount, expectedSuccessful)
			t.Logf("  Failed: %d (expected: %d)", failureCount, expectedFailures)
			t.Logf("  Final remaining credits: %d (expected: %d)", minRemainingCredits, expectedRemaining)

			// Verify exact accuracy - decrement should be 100% accurate
			require.Equal(t, totalRequests, successCount+failureCount, "All requests should be processed")
			require.Equal(t, expectedSuccessful, successCount, " decrement must be 100%% accurate for successes")
			require.Equal(t, expectedFailures, failureCount, " decrement must be 100%% accurate for failures")
			require.Equal(t, expectedRemaining, minRemainingCredits, "Remaining credits must be exactly accurate")

			// Verify traffic distribution in multi-node tests
			if tc.nodeCount > 1 {
				lbMetrics := lb.GetMetrics()
				require.Equal(t, tc.nodeCount, len(lbMetrics), "All nodes should have received traffic")

				for nodeID, count := range lbMetrics {
					percentage := float64(count) / float64(totalRequests) * 100
					require.LessOrEqual(t, percentage, 70.0,
						"Node %s handled %.1f%% of traffic (should be <= 70%%)", nodeID, percentage)
				}
			}

			// Verify final database consistency
			t.Logf("Waiting for database consistency...")

			var dbRemaining int64
			require.Eventually(t, func() bool {
				finalKey, err := db.Query.FindKeyByID(ctx, h.DB.RO(), keyResponse.KeyID)
				if err != nil {
					return false
				}

				if finalKey.RemainingRequests.Valid {
					dbRemaining = int64(finalKey.RemainingRequests.Int32)
					return dbRemaining == expectedRemaining
				}

				return false
			}, 10*time.Second, 200*time.Millisecond, "Database should reach exact consistency within timeout")

			t.Logf("Database remaining credits: %d", dbRemaining)

			// Database must also be 100% accurate
			require.Equal(t, expectedRemaining, dbRemaining,
				"Database remaining credits must be 100%% accurate after decrement")
		})
	}
}

// TestDecrementEdgeCases tests edge cases for the decrement logic
func TestDecrementEdgeCases(t *testing.T) {


	t.Run("ZeroCreditHandling", func(t *testing.T) {
		ctx := context.Background()
		h := integration.New(t, integration.Config{NumNodes: 1})

		workspace := h.Resources().UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:   ptr.P(int32(0)),
		})

		req := handler.Request{Key: keyResponse.Key}
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		lb := integration.NewLoadbalancer(h)

		// All requests should fail immediately
		for i := 0; i < 5; i++ {
			res, err := integration.CallRandomNode[handler.Request, handler.Response](
				lb, "POST", "/v2/keys.verifyKey", headers, req)

			require.NoError(t, err)
			require.Equal(t, 200, res.Status)
			require.False(t, res.Body.Data.Valid, "Request %d should fail with 0 credits", i+1)
		}
	})

	t.Run("ExactCreditDepletion", func(t *testing.T) {
		ctx := context.Background()
		h := integration.New(t, integration.Config{NumNodes: 1})

		workspace := h.Resources().UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:   ptr.P(int32(10)),
		})

		req := handler.Request{Key: keyResponse.Key}
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		lb := integration.NewLoadbalancer(h)

		// First 10 requests should succeed
		for i := 0; i < 10; i++ {
			res, err := integration.CallRandomNode[handler.Request, handler.Response](
				lb, "POST", "/v2/keys.verifyKey", headers, req)

			require.NoError(t, err)
			require.Equal(t, 200, res.Status)
			require.True(t, res.Body.Data.Valid, "Request %d should succeed", i+1)

			if res.Body.Data.Credits != nil {
				expectedRemaining := 10 - (i + 1)
				require.Equal(t, int32(expectedRemaining), *res.Body.Data.Credits,
					"Request %d should leave %d credits", i+1, expectedRemaining)
			}
		}

		// 11th request should fail
		res, err := integration.CallRandomNode[handler.Request, handler.Response](
			lb, "POST", "/v2/keys.verifyKey", headers, req)

		require.NoError(t, err)
		require.Equal(t, 200, res.Status)
		require.False(t, res.Body.Data.Valid, "11th request should fail")
	})

	t.Run("HighCostConcurrency", func(t *testing.T) {
		ctx := context.Background()
		h := integration.New(t, integration.Config{NumNodes: 3})

		workspace := h.Resources().UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		keyResponse := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:   ptr.P(int32(25)),
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		lb := integration.NewLoadbalancer(h)

		// Test concurrent high-cost requests
		const numGoroutines = 10
		const cost = 5

		results := make(chan bool, numGoroutines)
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := handler.Request{
					Key: keyResponse.Key,
					Credits: &openapi.KeysVerifyKeyCredits{
						Cost: int32(cost),
					},
				}

				res, err := integration.CallRandomNode[handler.Request, handler.Response](
					lb, "POST", "/v2/keys.verifyKey", headers, req)

				require.NoError(t, err)
				require.Equal(t, 200, res.Status)

				results <- res.Body.Data.Valid
			}()
		}

		wg.Wait()
		close(results)

		// Count successes - should be exactly 5 (5 * 5 = 25 credits)
		var successCount int
		for success := range results {
			if success {
				successCount++
			}
		}

		t.Logf("High cost concurrency: %d successes out of %d attempts", successCount, numGoroutines)
		require.Equal(t, 5, successCount, "should have exactly 5 successful high-cost requests")
	})
}
