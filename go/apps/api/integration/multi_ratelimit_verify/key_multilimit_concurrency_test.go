//go:build integration

package multi_ratelimit_verify

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// TestKeyWithMultipleRatelimits_HighConcurrency tests a single key with multiple rate limits
// under extreme concurrent load to verify:
// 1. Counter accuracy (no leaks when limits are exceeded)
// 2. Lock contention doesn't cause deadlocks
// 3. All-or-nothing semantics work correctly
func TestKeyWithMultipleRatelimits_HighConcurrency(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	t.Run("counter leak test - multi limits", func(t *testing.T) {
		// This test specifically checks that when one limit fails, NO counters are incremented
		ctx := context.Background()

		h := integration.New(t, integration.Config{
			NumNodes: 3,
		})

		workspace := h.Seed.Resources.UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		// Create key with two limits where one is very restrictive
		key := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "restrictive",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    3000, // 3 seconds
					Limit:       5,    // Only 5 allowed
				},
				{
					Name:        "permissive",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    3000, // 3 seconds
					Limit:       1000, // Very high
				},
			},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Key: key.Key,
		}

		lb := integration.NewLoadbalancer(h)

		// Make exactly 5 requests (should all succeed)
		for i := 0; i < 5; i++ {
			res, err := integration.CallRandomNode[handler.Request, handler.Response](
				lb, "POST", "/v2/keys.verifyKey", headers, req)

			require.NoError(t, err)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "request %d should be valid", i+1)
		}

		// Wait for replay buffer to sync
		time.Sleep(500 * time.Millisecond)

		// 6th request should be rate limited by restrictive limit
		res, err := integration.CallRandomNode[handler.Request, handler.Response](
			lb, "POST", "/v2/keys.verifyKey", headers, req)

		require.NoError(t, err)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "request 6 should be rate limited")

		// CRITICAL: Check the permissive limit's remaining count
		restrictiveRemaining := int64(-1)
		permissiveRemaining := int64(-1)

		if res.Body.Data.Ratelimits != nil {
			for _, rl := range *res.Body.Data.Ratelimits {
				if rl.Name == "restrictive" {
					restrictiveRemaining = rl.Remaining
				}
				if rl.Name == "permissive" {
					permissiveRemaining = rl.Remaining
				}
			}
		}

		require.Equal(t, int64(0), restrictiveRemaining, "restrictive limit should have 0 remaining")
		require.Equal(t, int64(995), permissiveRemaining,
			"permissive limit should have 995 remaining (1000 - 5). "+
				"If this is 994, the bug exists: the counter was incremented even though the request was rejected.")
	})

	t.Run("high concurrency burst - deadlock test", func(t *testing.T) {
		ctx := context.Background()

		h := integration.New(t, integration.Config{
			NumNodes: 3,
		})

		workspace := h.Seed.Resources.UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		// Create a key with very restrictive limits to force high contention
		key := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "limit-1",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2000, // 2 seconds
					Limit:       10,   // Very restrictive
				},
				{
					Name:        "limit-2",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2000,
					Limit:       10,
				},
				{
					Name:        "limit-3",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2000,
					Limit:       10,
				},
			},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Key: key.Key,
		}

		// Extreme burst: 100 requests at the exact same time
		concurrentRequests := 100

		t.Logf("Sending %d requests in parallel burst to test lock contention", concurrentRequests)

		var successCount atomic.Int64
		var rateLimitedCount atomic.Int64
		var wg sync.WaitGroup
		wg.Add(concurrentRequests)

		lb := integration.NewLoadbalancer(h)

		realStart := time.Now()

		// Send all requests at once
		for i := 0; i < concurrentRequests; i++ {
			go func() {
				defer wg.Done()

				res, err := integration.CallRandomNode[handler.Request, handler.Response](
					lb, "POST", "/v2/keys.verifyKey", headers, req)

				require.NoError(t, err)
				require.Equal(t, 200, res.Status)

				if res.Body.Data.Valid {
					successCount.Add(1)
				} else {
					rateLimitedCount.Add(1)
				}
			}()
		}

		// Wait with timeout to catch deadlocks
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			realDuration := time.Since(realStart)
			t.Logf("Burst completed in %s", realDuration)
			t.Logf("Results: %d valid, %d rate limited", successCount.Load(), rateLimitedCount.Load())

			// All 3 limits have limit=10, so at most 10 should succeed
			// Due to multi-node eventual consistency, allow some slack
			require.GreaterOrEqual(t, successCount.Load(), int64(10), "at least 10 should succeed")
			require.LessOrEqual(t, successCount.Load(), int64(30), "should not massively over-admit")

			// All requests should complete (no deadlocks)
			require.Equal(t, int64(concurrentRequests), successCount.Load()+rateLimitedCount.Load())

		case <-time.After(10 * time.Second):
			t.Fatal("Test timed out - likely deadlock in RatelimitMany")
		}
	})

	t.Run("sustained load - stress test", func(t *testing.T) {
		// Test sustained concurrent load to see performance degradation
		ctx := context.Background()

		h := integration.New(t, integration.Config{
			NumNodes: 5,
		})

		workspace := h.Seed.Resources.UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		key := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "limit-1s",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    1000, // 1 second
					Limit:       50,
				},
				{
					Name:        "limit-2s",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2000, // 2 seconds
					Limit:       90,
				},
				{
					Name:        "limit-5s",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    5000, // 5 seconds
					Limit:       200,
				},
			},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Key: key.Key,
		}

		workers := 50
		requestsPerWorker := 10
		totalRequests := workers * requestsPerWorker

		t.Logf("Sending %d requests (%d workers × %d each) with 3 rate limits per key",
			totalRequests, workers, requestsPerWorker)

		var successCount atomic.Int64
		var rateLimitedCount atomic.Int64
		var wg sync.WaitGroup
		wg.Add(workers)

		lb := integration.NewLoadbalancer(h)
		start := time.Now()

		for w := 0; w < workers; w++ {
			go func() {
				defer wg.Done()

				for i := 0; i < requestsPerWorker; i++ {
					time.Sleep(time.Millisecond * 20) // Spread load over time

					res, err := integration.CallRandomNode[handler.Request, handler.Response](
						lb, "POST", "/v2/keys.verifyKey", headers, req)

					require.NoError(t, err)

					if res.Body.Data.Valid {
						successCount.Add(1)
					} else {
						rateLimitedCount.Add(1)
					}
				}
			}()
		}

		wg.Wait()

		duration := time.Since(start)
		t.Logf("Completed in %s", duration)
		t.Logf("Results: %d valid, %d rate limited", successCount.Load(), rateLimitedCount.Load())

		// Verify no deadlocks
		require.Equal(t, int64(totalRequests), successCount.Load()+rateLimitedCount.Load())

		// Verify all nodes received traffic
		lbMetrics := lb.GetMetrics()
		require.Equal(t, 5, len(lbMetrics), "all 5 nodes should have received traffic")
	})
}
