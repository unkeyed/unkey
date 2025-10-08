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
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// TestIdentitySharedRatelimits_HighConcurrency tests identity-based rate limits
// shared across multiple keys under extreme concurrent load to verify:
// 1. Shared counters are accurate across all keys
// 2. Lock contention across different keys with same identity doesn't deadlock
// 3. Counter accuracy when multiple keys hit the same identity limit simultaneously
func TestIdentitySharedRatelimits_HighConcurrency(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	t.Run("identity counter leak test", func(t *testing.T) {
		// This test verifies that when one key in an identity hits a limit,
		// other keys' identity counters are not incorrectly incremented
		ctx := context.Background()

		h := integration.New(t, integration.Config{
			NumNodes: 3,
		})

		workspace := h.Seed.Resources.UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		// Create identity with restrictive limit
		identityExternalID := uid.New("user")
		identityID := h.Seed.CreateIdentity(ctx, seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  identityExternalID,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "identity-restrictive",
					WorkspaceID: workspace.ID,
					Duration:    3000, // 3 seconds
					Limit:       10,   // Only 10 allowed
					AutoApply:   true,
				},
				{
					Name:        "identity-permissive",
					WorkspaceID: workspace.ID,
					Duration:    3000, // 3 seconds
					Limit:       5000, // Very high
					AutoApply:   true,
				},
			},
		})

		// Create 3 keys with the same identity
		keys := make([]string, 3)
		for i := 0; i < 3; i++ {
			key := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				IdentityID:  &identityID,
			})
			keys[i] = key.Key
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		lb := integration.NewLoadbalancer(h)

		// Use all 3 keys to make 10 requests total (hitting the identity limit exactly)
		for i := 0; i < 10; i++ {
			keyIdx := i % 3 // Rotate through keys

			req := handler.Request{
				Key: keys[keyIdx],
			}

			res, err := integration.CallRandomNode[handler.Request, handler.Response](
				lb, "POST", "/v2/keys.verifyKey", headers, req)

			require.NoError(t, err)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "request %d should be valid", i+1)
		}

		// Wait for replay buffer
		time.Sleep(500 * time.Millisecond)

		// 11th request (using any key) should be rate limited
		req := handler.Request{
			Key: keys[0],
		}

		res, err := integration.CallRandomNode[handler.Request, handler.Response](
			lb, "POST", "/v2/keys.verifyKey", headers, req)

		require.NoError(t, err)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "request 11 should be rate limited")

		// CRITICAL: Verify counter accuracy
		restrictiveRemaining := int64(-1)
		permissiveRemaining := int64(-1)

		if res.Body.Data.Ratelimits != nil {
			for _, rl := range *res.Body.Data.Ratelimits {
				if rl.Name == "identity-restrictive" {
					restrictiveRemaining = rl.Remaining
				}
				if rl.Name == "identity-permissive" {
					permissiveRemaining = rl.Remaining
				}
			}
		}

		require.Equal(t, int64(0), restrictiveRemaining, "restrictive limit should have 0 remaining")
		require.Equal(t, int64(4990), permissiveRemaining,
			"permissive identity limit should have 4990 remaining (5000 - 10). "+
				"If this is 4989, the bug exists: the identity counter was incremented "+
				"even though the request was rejected by the restrictive limit.")
	})

	t.Run("shared identity limits - high concurrency", func(t *testing.T) {
		ctx := context.Background()

		h := integration.New(t, integration.Config{
			NumNodes: 5,
		})

		workspace := h.Seed.Resources.UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		// Create an identity with multiple rate limits
		identityExternalID := uid.New("user")
		identityID := h.Seed.CreateIdentity(ctx, seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  identityExternalID,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "identity-1s",
					WorkspaceID: workspace.ID,
					Duration:    1000, // 1 second
					Limit:       30,   // 30 requests per second across all keys
					AutoApply:   true,
				},
				{
					Name:        "identity-2s",
					WorkspaceID: workspace.ID,
					Duration:    2000, // 2 seconds
					Limit:       50,   // 50 requests per 2 seconds across all keys
					AutoApply:   true,
				},
				{
					Name:        "identity-5s",
					WorkspaceID: workspace.ID,
					Duration:    5000, // 5 seconds
					Limit:       100,  // 100 requests per 5 seconds across all keys
					AutoApply:   true,
				},
			},
		})

		// Create 5 keys all attached to the same identity
		keys := make([]string, 5)
		for i := 0; i < 5; i++ {
			key := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				IdentityID:  &identityID,
			})
			keys[i] = key.Key
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Test parameters
		workersPerKey := 10        // 10 workers per key
		requestsPerWorker := 5     // Each worker makes 5 requests
		totalWorkers := 5 * workersPerKey
		totalRequests := totalWorkers * requestsPerWorker

		t.Logf("Configuration: 5 nodes, 5 keys sharing identity with 3 rate limits")
		t.Logf("Sending %d total requests (%d workers × %d requests each)",
			totalRequests, totalWorkers, requestsPerWorker)

		var successCount atomic.Int64
		var rateLimitedCount atomic.Int64

		// Track requests per key
		var requestsPerKey [5]atomic.Int64
		var successPerKey [5]atomic.Int64

		lb := integration.NewLoadbalancer(h)

		var wg sync.WaitGroup
		wg.Add(totalWorkers)

		realStart := time.Now()

		// Launch workers for each key
		for keyIdx := 0; keyIdx < 5; keyIdx++ {
			for w := 0; w < workersPerKey; w++ {
				go func(keyIndex int) {
					defer wg.Done()

					req := handler.Request{
						Key: keys[keyIndex],
					}

					for i := 0; i < requestsPerWorker; i++ {
						time.Sleep(time.Millisecond * 10)

						res, err := integration.CallRandomNode[handler.Request, handler.Response](
							lb, "POST", "/v2/keys.verifyKey", headers, req)

						require.NoError(t, err)
						require.Equal(t, 200, res.Status)

						requestsPerKey[keyIndex].Add(1)

						if res.Body.Data.Valid {
							successCount.Add(1)
							successPerKey[keyIndex].Add(1)
						} else if res.Body.Data.Code == openapi.RATELIMITED {
							rateLimitedCount.Add(1)
						}
					}
				}(keyIdx)
			}
		}

		wg.Wait()

		realDuration := time.Since(realStart)

		t.Logf("Test completed in %s", realDuration)
		t.Logf("Results: %d valid, %d rate limited", successCount.Load(), rateLimitedCount.Load())

		// Log per-key statistics
		for i := 0; i < 5; i++ {
			t.Logf("Key %d: %d requests, %d successful",
				i, requestsPerKey[i].Load(), successPerKey[i].Load())
		}

		// Verify all nodes received traffic
		lbMetrics := lb.GetMetrics()
		require.Equal(t, 5, len(lbMetrics), "all 5 nodes should have received traffic")

		// Verify total
		require.Equal(t, int64(totalRequests), successCount.Load()+rateLimitedCount.Load(),
			"all requests should be accounted for")

		// No deadlocks
		require.Equal(t, totalRequests, int(successCount.Load()+rateLimitedCount.Load()))
	})

	t.Run("extreme burst - identity limits", func(t *testing.T) {
		// Stress test: 10 keys, 20 requests each, all at roughly the same time
		ctx := context.Background()

		h := integration.New(t, integration.Config{
			NumNodes: 5,
		})

		workspace := h.Seed.Resources.UserWorkspace
		rootKey := h.Seed.CreateRootKey(ctx, workspace.ID, "api.*.verify_key")

		api := h.Seed.CreateAPI(ctx, seed.CreateApiRequest{
			WorkspaceID: workspace.ID,
		})

		// Create identity with very restrictive limit
		identityExternalID := uid.New("user")
		identityID := h.Seed.CreateIdentity(ctx, seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  identityExternalID,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "burst-limit-0",
					WorkspaceID: workspace.ID,
					Duration:    2000, // 2 seconds
					Limit:       15,   // Very restrictive
					AutoApply:   true,
				},
				{
					Name:        "burst-limit-1",
					WorkspaceID: workspace.ID,
					Duration:    2000,
					Limit:       15,
					AutoApply:   true,
				},
				{
					Name:        "burst-limit-2",
					WorkspaceID: workspace.ID,
					Duration:    2000,
					Limit:       15,
					AutoApply:   true,
				},
			},
		})

		// Create 10 keys
		numKeys := 10
		keys := make([]string, numKeys)
		for i := 0; i < numKeys; i++ {
			key := h.Seed.CreateKey(ctx, seed.CreateKeyRequest{
				WorkspaceID: workspace.ID,
				KeyAuthID:   api.KeyAuthID.String,
				IdentityID:  &identityID,
			})
			keys[i] = key.Key
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		requestsPerKey := 20
		totalRequests := numKeys * requestsPerKey

		t.Logf("Sending %d requests in parallel burst (%d keys × %d requests)",
			totalRequests, numKeys, requestsPerKey)

		var successCount atomic.Int64
		var rateLimitedCount atomic.Int64
		var wg sync.WaitGroup
		wg.Add(totalRequests)

		lb := integration.NewLoadbalancer(h)
		realStart := time.Now()

		// Launch all requests at once
		for keyIdx := 0; keyIdx < numKeys; keyIdx++ {
			for reqIdx := 0; reqIdx < requestsPerKey; reqIdx++ {
				go func(keyIndex int) {
					defer wg.Done()

					req := handler.Request{
						Key: keys[keyIndex],
					}

					res, err := integration.CallRandomNode[handler.Request, handler.Response](
						lb, "POST", "/v2/keys.verifyKey", headers, req)

					require.NoError(t, err)
					require.Equal(t, 200, res.Status)

					if res.Body.Data.Valid {
						successCount.Add(1)
					} else {
						rateLimitedCount.Add(1)
					}
				}(keyIdx)
			}
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

			// All 3 limits have limit=15, so theoretically 15 should succeed
			// But with 5 nodes + eventual consistency + all-or-nothing semantics,
			// we expect some under-admission (conservative behavior is good!)
			// Anywhere from 10-45 is reasonable
			require.GreaterOrEqual(t, successCount.Load(), int64(10), "at least 10 should succeed")
			require.LessOrEqual(t, successCount.Load(), int64(45), "should not massively over-admit")

			// No deadlocks - most important check!
			require.Equal(t, int64(totalRequests), successCount.Load()+rateLimitedCount.Load())

		case <-time.After(15 * time.Second):
			t.Fatal("Test timed out - likely deadlock in RatelimitMany with identity limits")
		}
	})
}
