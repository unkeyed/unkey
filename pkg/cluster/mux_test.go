package cluster

import (
	"testing"

	"github.com/stretchr/testify/require"
	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
)

func cacheInvalidationMessage(cacheName, cacheKey string) *clusterv1.ClusterMessage {
	return &clusterv1.ClusterMessage{
		Payload: &clusterv1.ClusterMessage_CacheInvalidation{
			CacheInvalidation: &cachev1.CacheInvalidationEvent{
				CacheName: cacheName,
				Action:    &cachev1.CacheInvalidationEvent_CacheKey{CacheKey: cacheKey},
			},
		},
	}
}

func TestMessageMux_RoutesToSubscriber(t *testing.T) {
	t.Run("delivers payload to typed subscriber", func(t *testing.T) {
		mux := NewMessageMux()

		var received *cachev1.CacheInvalidationEvent
		Subscribe(mux, func(payload *clusterv1.ClusterMessage_CacheInvalidation) {
			received = payload.CacheInvalidation
		})

		msg := cacheInvalidationMessage("my-cache", "my-key")
		mux.OnMessage(msg)

		require.NotNil(t, received)
		require.Equal(t, "my-cache", received.GetCacheName())
		require.Equal(t, "my-key", received.GetCacheKey())
	})
}

func TestMessageMux_MultipleSubscribers(t *testing.T) {
	t.Run("fans out to all subscribers", func(t *testing.T) {
		mux := NewMessageMux()

		var count1, count2 int
		Subscribe(mux, func(payload *clusterv1.ClusterMessage_CacheInvalidation) { count1++ })
		Subscribe(mux, func(payload *clusterv1.ClusterMessage_CacheInvalidation) { count2++ })

		mux.OnMessage(cacheInvalidationMessage("c", "k"))

		require.Equal(t, 1, count1)
		require.Equal(t, 1, count2)
	})
}

func TestMessageMux_NoSubscribersNoOp(t *testing.T) {
	t.Run("no panic without subscribers", func(t *testing.T) {
		mux := NewMessageMux()

		// Should not panic when no subscribers are registered
		mux.OnMessage(cacheInvalidationMessage("c", "k"))
	})
}
