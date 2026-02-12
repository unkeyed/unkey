package cluster

import (
	"testing"

	cachev1 "github.com/unkeyed/unkey/gen/proto/cache/v1"
	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func marshalEnvelope(t *testing.T, msg *clusterv1.ClusterMessage) []byte {
	t.Helper()
	data, err := proto.Marshal(msg)
	require.NoError(t, err)
	return data
}

func cacheInvalidationEnvelope(cacheName, cacheKey string) *clusterv1.ClusterMessage {
	return &clusterv1.ClusterMessage{
		Message: &clusterv1.ClusterMessage_CacheInvalidation{
			CacheInvalidation: &cachev1.CacheInvalidationEvent{
				CacheName: cacheName,
				CacheKey:  cacheKey,
			},
		},
	}
}

func TestMessageMux_RoutesToCorrectHandler(t *testing.T) {
	mux := NewMessageMux()

	var received *cachev1.CacheInvalidationEvent
	mux.HandleCacheInvalidation(func(event *cachev1.CacheInvalidationEvent) {
		received = event
	})

	msg := marshalEnvelope(t, cacheInvalidationEnvelope("my-cache", "my-key"))
	mux.OnMessage(msg)

	require.NotNil(t, received)
	require.Equal(t, "my-cache", received.CacheName)
	require.Equal(t, "my-key", received.CacheKey)
}

func TestMessageMux_NoHandlerDropped(t *testing.T) {
	mux := NewMessageMux()

	msg := marshalEnvelope(t, cacheInvalidationEnvelope("c", "k"))

	// Should not panic when no handler is registered
	mux.OnMessage(msg)
}

func TestMessageMux_MalformedMessageDropped(t *testing.T) {
	mux := NewMessageMux()

	mux.HandleCacheInvalidation(func(_ *cachev1.CacheInvalidationEvent) {
		t.Fatal("handler should not be called for malformed message")
	})

	// Should not panic
	mux.OnMessage([]byte("this is not valid protobuf"))
}

func TestWrap(t *testing.T) {
	envelope := cacheInvalidationEnvelope("test-cache", "test-key")

	wrapped, err := Wrap(envelope)
	require.NoError(t, err)

	var decoded clusterv1.ClusterMessage
	err = proto.Unmarshal(wrapped, &decoded)
	require.NoError(t, err)

	ci := decoded.GetCacheInvalidation()
	require.NotNil(t, ci)
	require.Equal(t, "test-cache", ci.CacheName)
	require.Equal(t, "test-key", ci.CacheKey)
}
