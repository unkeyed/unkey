package cluster

import (
	"testing"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/stretchr/testify/require"
)

func cacheInvalidationMessage(cacheName, cacheKey string) *clusterv1.ClusterMessage {
	return &clusterv1.ClusterMessage{
		Message: &clusterv1.ClusterMessage_CacheInvalidation{
			CacheInvalidation: &cachev1.CacheInvalidationEvent{
				CacheName: cacheName,
				Action:    &cachev1.CacheInvalidationEvent_CacheKey{CacheKey: cacheKey},
			},
		},
	}
}

func TestMessageMux_RoutesToSubscriber(t *testing.T) {
	mux := NewMessageMux()

	var received *clusterv1.ClusterMessage
	mux.Subscribe(func(msg *clusterv1.ClusterMessage) {
		received = msg
	})

	msg := cacheInvalidationMessage("my-cache", "my-key")
	mux.OnMessage(msg)

	require.NotNil(t, received)
	require.Equal(t, "my-cache", received.GetCacheInvalidation().GetCacheName())
	require.Equal(t, "my-key", received.GetCacheInvalidation().GetCacheKey())
}

func TestMessageMux_MultipleSubscribers(t *testing.T) {
	mux := NewMessageMux()

	var count1, count2 int
	mux.Subscribe(func(msg *clusterv1.ClusterMessage) { count1++ })
	mux.Subscribe(func(msg *clusterv1.ClusterMessage) { count2++ })

	mux.OnMessage(cacheInvalidationMessage("c", "k"))

	require.Equal(t, 1, count1)
	require.Equal(t, 1, count2)
}

func TestMessageMux_NoSubscribersNoOp(t *testing.T) {
	mux := NewMessageMux()

	// Should not panic when no subscribers are registered
	mux.OnMessage(cacheInvalidationMessage("c", "k"))
}
