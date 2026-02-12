package cluster

import (
	"sync/atomic"
	"testing"

	clusterv1 "github.com/unkeyed/unkey/gen/proto/cluster/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func marshalEnvelope(t *testing.T, msgType string, payload []byte) []byte {
	t.Helper()
	data, err := proto.Marshal(&clusterv1.ClusterMessage{
		Type:    msgType,
		Payload: payload,
	})
	require.NoError(t, err)
	return data
}

func TestMessageMux_RoutesToCorrectHandler(t *testing.T) {
	mux := NewMessageMux()

	var received []byte
	mux.Handle("cache.invalidation", func(payload []byte) {
		received = payload
	})

	innerPayload := []byte("hello-cache")
	msg := marshalEnvelope(t, "cache.invalidation", innerPayload)

	mux.OnMessage(msg)

	require.Equal(t, innerPayload, received)
}

func TestMessageMux_UnknownTypeDropped(t *testing.T) {
	mux := NewMessageMux()

	msg := marshalEnvelope(t, "unknown.type", []byte("data"))

	// Should not panic
	mux.OnMessage(msg)
}

func TestMessageMux_MultipleHandlers(t *testing.T) {
	mux := NewMessageMux()

	var cacheCount atomic.Int32
	var ratelimitCount atomic.Int32

	mux.Handle("cache.invalidation", func(_ []byte) {
		cacheCount.Add(1)
	})
	mux.Handle("ratelimit.sync", func(_ []byte) {
		ratelimitCount.Add(1)
	})

	mux.OnMessage(marshalEnvelope(t, "cache.invalidation", []byte("c1")))
	mux.OnMessage(marshalEnvelope(t, "ratelimit.sync", []byte("r1")))
	mux.OnMessage(marshalEnvelope(t, "cache.invalidation", []byte("c2")))

	require.Equal(t, int32(2), cacheCount.Load())
	require.Equal(t, int32(1), ratelimitCount.Load())
}

func TestMessageMux_MalformedMessageDropped(t *testing.T) {
	mux := NewMessageMux()

	mux.Handle("cache.invalidation", func(_ []byte) {
		t.Fatal("handler should not be called for malformed message")
	})

	// Should not panic
	mux.OnMessage([]byte("this is not valid protobuf"))
}

func TestWrap(t *testing.T) {
	payload := []byte("inner-data")

	wrapped, err := Wrap("cache.invalidation", payload)
	require.NoError(t, err)

	var envelope clusterv1.ClusterMessage
	err = proto.Unmarshal(wrapped, &envelope)
	require.NoError(t, err)

	require.Equal(t, "cache.invalidation", envelope.Type)
	require.Equal(t, payload, envelope.Payload)
}
