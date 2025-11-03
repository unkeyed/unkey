package cache

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
	cachev1 "github.com/unkeyed/unkey/go/gen/proto/cache/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestAPI_ProducesInvalidationEvents(t *testing.T) {
	testutil.SkipUnlessIntegration(t)

	// Set up event stream listener to capture invalidation events BEFORE starting API node
	brokers := containers.Kafka(t)
	topicName := "cache-invalidations" // Use same topic as API nodes

	// Create topic
	topic, err := eventstream.NewTopic[*cachev1.CacheInvalidationEvent](eventstream.TopicConfig{
		Brokers:    brokers,
		Topic:      topicName,
		InstanceID: uid.New(uid.TestPrefix),
		Logger:     logging.NewNoop(),
	})
	require.NoError(t, err)

	// Ensure topic exists
	err = topic.EnsureExists(1, 1)
	require.NoError(t, err, "Should be able to create topic")
	defer topic.Close()

	// Track received events
	var receivedEvents []*cachev1.CacheInvalidationEvent
	var eventsMutex sync.Mutex

	// Start consumer from latest offset (not beginning) to avoid old test events
	consumer := topic.NewConsumer()
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	consumer.Consume(ctx, func(ctx context.Context, event *cachev1.CacheInvalidationEvent) error {
		eventsMutex.Lock()
		receivedEvents = append(receivedEvents, event)
		eventsMutex.Unlock()

		return nil
	})

	// Wait for consumer to be ready and positioned at latest offset
	time.Sleep(5 * time.Second)

	// Now start API node AFTER consumer is ready at latest offset
	h := integration.New(t, integration.Config{NumNodes: 1})
	addr := h.GetClusterAddrs()[0]

	// Create test API
	api := h.Seed.CreateAPI(context.Background(), seed.CreateApiRequest{
		WorkspaceID: h.Seed.Resources.UserWorkspace.ID,
	})
	rootKey := h.Seed.CreateRootKey(context.Background(), api.WorkspaceID, fmt.Sprintf("api.%s.read_api", api.ID), fmt.Sprintf("api.%s.delete_api", api.ID))

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Test 1: API deletion should produce cache invalidation events
	_, err = integration.CallNode[openapi.V2ApisDeleteApiRequestBody, openapi.V2ApisDeleteApiResponseBody](
		t, addr, "POST", "/v2/apis.deleteApi",
		headers,
		openapi.V2ApisDeleteApiRequestBody{ApiId: api.ID},
	)
	require.NoError(t, err, "API deletion should succeed")

	// Wait for invalidation events to be produced
	require.Eventually(t, func() bool {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()
		return len(receivedEvents) > 0
	}, 15*time.Second, 200*time.Millisecond, "API deletion should produce cache invalidation events within 15 seconds")

	// Verify events
	eventsMutex.Lock()
	defer eventsMutex.Unlock()

	require.Greater(t, len(receivedEvents), 0, "Should receive at least one invalidation event")

	// Log all received events for debugging
	t.Logf("Received %d invalidation events:", len(receivedEvents))
	for i, event := range receivedEvents {
		t.Logf("  Event %d: CacheName=%s, CacheKey=%s, SourceInstance=%s",
			i, event.CacheName, event.CacheKey, event.SourceInstance)
	}

	// Look for live_api_by_id cache invalidation event
	// The cache key is scoped with format "workspaceID:apiID"
	expectedCacheKey := cache.ScopedKey{
		WorkspaceID: api.WorkspaceID,
		Key:         api.ID,
	}.String()
	var apiByIdEvent *cachev1.CacheInvalidationEvent
	for _, event := range receivedEvents {
		if event.CacheName == "live_api_by_id" && event.CacheKey == expectedCacheKey {
			apiByIdEvent = event
			break
		}
	}

	t.Logf("Looking for cache key: %s", expectedCacheKey)

	require.NotNil(t, apiByIdEvent, "Should receive live_api_by_id invalidation event")
	require.Equal(t, "live_api_by_id", apiByIdEvent.CacheName, "Event should be for live_api_by_id cache")
	require.Equal(t, expectedCacheKey, apiByIdEvent.CacheKey, "Event should be for correct scoped cache key")
	require.NotEmpty(t, apiByIdEvent.SourceInstance, "Event should have source instance")
	require.Greater(t, apiByIdEvent.Timestamp, int64(0), "Event should have valid timestamp")
}
