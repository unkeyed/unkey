package cache

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/integration"
	"github.com/unkeyed/unkey/apps/api/openapi"
	"github.com/unkeyed/unkey/pkg/debug"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
)

func TestDistributedCacheInvalidation_EndToEnd(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	// Start a 3-node cluster
	h := integration.New(t, integration.Config{NumNodes: 3})
	clusterAddrs := h.GetClusterAddrs()
	require.Len(t, clusterAddrs, 3, "Should have 3 nodes in cluster")

	// Create test API
	api := h.Seed.CreateAPI(context.Background(), seed.CreateApiRequest{
		WorkspaceID: h.Seed.Resources.UserWorkspace.ID,
	})
	rootKey := h.Seed.CreateRootKey(context.Background(), api.WorkspaceID, fmt.Sprintf("api.%s.read_api", api.ID), fmt.Sprintf("api.%s.delete_api", api.ID))

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Step 1: Populate cache on all nodes by making API calls
	for i, addr := range clusterAddrs {
		resp, err := integration.CallNode[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
			t, addr, "POST", "/v2/apis.getApi",
			headers,
			openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
		)
		require.NoError(t, err, "Node %d should respond to API call", i)
		require.Equal(t, http.StatusOK, resp.Status, "API should exist on node %d", i)
		require.Equal(t, api.ID, resp.Body.Data.Id, "API ID should match on node %d", i)
		require.Equal(t, api.Name, resp.Body.Data.Name, "API name should match on node %d", i)

		// Verify cache is populated (should show FRESH or MISS on first call)
		cacheHeaders := resp.Headers.Values("X-Unkey-Debug-Cache")
		require.NotEmpty(t, cacheHeaders, "Node %d should have cache debug headers", i)
	}

	// Step 2: Delete API from first node (this should trigger distributed invalidation)
	deleteResp, err := integration.CallNode[openapi.V2ApisDeleteApiRequestBody, openapi.V2ApisDeleteApiResponseBody](
		t, clusterAddrs[0], "POST", "/v2/apis.deleteApi",
		headers,
		openapi.V2ApisDeleteApiRequestBody{ApiId: api.ID},
	)
	require.NoError(t, err, "API deletion should succeed on node 0")
	require.Equal(t, http.StatusOK, deleteResp.Status, "API deletion should return 200 on node 0")

	// Step 3: Verify cache invalidation propagates to all nodes within 5 seconds
	for i, addr := range clusterAddrs {
		var nodeInvalidated atomic.Bool

		require.Eventually(t, func() bool {
			// Try to get the deleted API
			resp, err := integration.CallNode[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
				t, addr, "POST", "/v2/apis.getApi",
				headers,
				openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
			)
			if err != nil {
				t.Logf("Node %d: Error calling API: %v", i, err)
				return false
			}

			t.Logf("Node %d: API call status: %d", i, resp.Status)

			// API should be deleted (404)
			if resp.Status != http.StatusNotFound {
				// Debug: log what we got instead of 404
				if resp.Status == http.StatusOK {
					t.Logf("Node %d: API still exists - cache may not be invalidated yet", i)
				} else {
					t.Logf("Node %d: Unexpected status %d (expected 404)", i, resp.Status)
				}
				return false
			}

			// If we got 404, the distributed cache invalidation worked!
			// The cache could show different statuses:
			// - MISS: cache entry was evicted
			// - FRESH/HIT: cache was updated with "deleted" state
			// Either way, the important thing is that the API correctly returns 404
			t.Logf("Node %d: API correctly returns 404 - distributed cache invalidation worked!", i)

			// Log cache headers for debugging, but don't require specific status
			cacheHeaders := resp.Headers.Values("X-Unkey-Debug-Cache")
			for _, headerValue := range cacheHeaders {
				parsedHeader, err := debug.ParseCacheHeader(headerValue)
				if err == nil && parsedHeader.CacheName == "live_api_by_id" {
					t.Logf("Node %d: api_by_id cache status: %s (latency: %v)", i, parsedHeader.Status, parsedHeader.Latency)
				}
			}

			nodeInvalidated.Store(true)
			return true
		}, 15*time.Second, 200*time.Millisecond, "Node %d should show API as deleted with cache invalidated within 15 seconds", i)

		require.True(t, nodeInvalidated.Load(), "Node %d should have invalidated cache after distributed event", i)
	}

	// Step 4: Verify all nodes consistently return 404 for the deleted API
	for i, addr := range clusterAddrs {
		resp, err := integration.CallNode[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
			t, addr, "POST", "/v2/apis.getApi",
			headers,
			openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
		)
		require.NoError(t, err, "Node %d should handle API call", i)
		require.Equal(t, http.StatusNotFound, resp.Status, "Node %d should return 404 for deleted API", i)
	}
}

func TestCacheDebugHeaders(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	// Start a single node for this test
	h := integration.New(t, integration.Config{NumNodes: 1})
	addr := h.GetClusterAddrs()[0]

	// Create test API
	api := h.Seed.CreateAPI(context.Background(), seed.CreateApiRequest{
		WorkspaceID: h.Seed.Resources.UserWorkspace.ID,
	})
	rootKey := h.Seed.CreateRootKey(context.Background(), api.WorkspaceID, fmt.Sprintf("api.%s.read_api", api.ID))

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Test cache debug headers format
	resp, err := integration.CallNode[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
		t, addr, "POST", "/v2/apis.getApi",
		headers,
		openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
	)
	require.NoError(t, err, "API call should succeed")
	require.Equal(t, http.StatusOK, resp.Status, "API should exist")

	// Verify cache debug headers format
	cacheHeaders := resp.Headers.Values("X-Unkey-Debug-Cache")
	require.NotEmpty(t, cacheHeaders, "Should have X-Unkey-Debug-Cache headers")

	// Each header should be parseable as a structured cache header
	for _, headerValue := range cacheHeaders {
		parsedHeader, err := debug.ParseCacheHeader(headerValue)
		require.NoError(t, err, "Cache header '%s' should be valid structured format", headerValue)

		// Validate components
		require.NotEmpty(t, parsedHeader.CacheName, "Cache name should not be empty")
		require.Positive(t, parsedHeader.Latency, "Latency should be positive")

		// Status should be one of the valid values
		validStatuses := []string{"FRESH", "STALE", "MISS", "ERROR"}
		statusValid := false
		for _, validStatus := range validStatuses {
			if parsedHeader.Status == validStatus {
				statusValid = true
				break
			}
		}
		require.True(t, statusValid,
			"Status '%s' should be one of FRESH, STALE, MISS, ERROR", parsedHeader.Status)

		t.Logf("Parsed cache header: cache=%s, latency=%v, status=%s",
			parsedHeader.CacheName, parsedHeader.Latency, parsedHeader.Status)
	}

	// Should have api_by_id cache entry
	foundApiCache := false
	for _, headerValue := range cacheHeaders {
		parsedHeader, err := debug.ParseCacheHeader(headerValue)
		if err != nil {
			continue // Already validated above, but be defensive
		}
		if parsedHeader.CacheName == "live_api_by_id" {
			foundApiCache = true
			t.Logf("Found api_by_id cache entry: latency=%v, status=%s",
				parsedHeader.Latency, parsedHeader.Status)
			break
		}
	}
	require.True(t, foundApiCache, "Should have api_by_id cache entry in debug headers")
}
