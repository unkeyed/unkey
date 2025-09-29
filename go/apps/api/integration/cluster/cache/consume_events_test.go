package cache

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/integration"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/debug"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestAPI_ConsumesInvalidationEvents(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	// Start a single API node
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

	// Step 1: Populate cache by making API call (first call will be MISS)
	resp, err := integration.CallNode[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
		t, addr, "POST", "/v2/apis.getApi",
		headers,
		openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
	)
	require.NoError(t, err, "Initial API call should succeed")
	require.Equal(t, http.StatusOK, resp.Status, "API should exist initially")

	// Step 1.5: Make a second call to populate cache (should be FRESH)
	resp2, err := integration.CallNode[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
		t, addr, "POST", "/v2/apis.getApi",
		headers,
		openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
	)
	require.NoError(t, err, "Second API call should succeed")
	require.Equal(t, http.StatusOK, resp2.Status, "API should exist on second call")

	// Verify cache shows fresh data in debug headers
	cacheHeaders := resp2.Headers.Values("X-Unkey-Debug-Cache")
	require.NotEmpty(t, cacheHeaders, "Should have cache debug headers")

	// Look for api_by_id cache with FRESH status
	foundFresh := false
	for _, headerValue := range cacheHeaders {
		parsedHeader, err := debug.ParseCacheHeader(headerValue)
		if err != nil {
			continue // Skip invalid headers
		}
		if parsedHeader.CacheName == "api_by_id" && parsedHeader.Status == "FRESH" {
			foundFresh = true
			t.Logf("Found FRESH cache: cache=%s, latency=%v, status=%s", parsedHeader.CacheName, parsedHeader.Latency, parsedHeader.Status)
			break
		}
	}
	require.True(t, foundFresh, "Cache should show FRESH status for api_by_id on second call")

	// Step 2: Produce invalidation event externally (simulating another node's action)
	brokers := containers.Kafka(t)
	topicName := "cache-invalidations"

	topic := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: uid.New(uid.TestPrefix), // Use unique ID to avoid conflicts with API node
		Logger:     logging.NewNoop(),
	})

	// Ensure topic exists before producing
	err = topic.EnsureExists(1)
	require.NoError(t, err, "Should be able to create topic")
	defer topic.Close()

	producer := topic.NewProducer()

	// Send invalidation event for the API
	invalidationEvent := &cachev1.CacheInvalidationEvent{
		CacheName:      "api_by_id",
		CacheKey:       api.ID,
		Timestamp:      time.Now().UnixMilli(),
		SourceInstance: "external-node",
	}

	ctx := context.Background()
	err = producer.Produce(ctx, invalidationEvent)
	require.NoError(t, err, "Should be able to produce invalidation event")

	// Step 3: Verify that the API node processes the invalidation and cache shows MISS/stale
	var cacheInvalidated atomic.Bool

	require.Eventually(t, func() bool {
		resp, err := integration.CallNode[openapi.V2ApisGetApiRequestBody, openapi.V2ApisGetApiResponseBody](
			t, addr, "POST", "/v2/apis.getApi",
			headers,
			openapi.V2ApisGetApiRequestBody{ApiId: api.ID},
		)

		if err != nil {
			return false
		}

		// Check cache debug headers for invalidation
		cacheHeaders := resp.Headers.Values("X-Unkey-Debug-Cache")
		if len(cacheHeaders) == 0 {
			return false
		}

		// Look for api_by_id cache that's no longer FRESH (should be MISS or STALE)
		for _, headerValue := range cacheHeaders {
			parsedHeader, err := debug.ParseCacheHeader(headerValue)
			if err != nil {
				continue // Skip invalid headers
			}
			if parsedHeader.CacheName == "api_by_id" {
				// Cache should no longer be FRESH after invalidation
				if parsedHeader.Status != "FRESH" {
					t.Logf("Found invalidated cache: cache=%s, latency=%v, status=%s",
						parsedHeader.CacheName, parsedHeader.Latency, parsedHeader.Status)
					cacheInvalidated.Store(true)
					return true
				}
			}
		}

		return false
	}, 15*time.Second, 200*time.Millisecond, "API node should process invalidation event and cache should no longer be FRESH within 15 seconds")

	require.True(t, cacheInvalidated.Load(), "Cache should be invalidated after receiving external invalidation event")
}
