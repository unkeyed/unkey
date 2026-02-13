package clustering_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/clustering"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/cluster"
)

// twoNodeCluster sets up a two-node gossip cluster with a ClusterCache on each node.
// Both caches share the same cache name ("test_cache") so invalidation events route correctly.
type twoNodeCluster struct {
	Cache1 *clustering.ClusterCache[string, string]
	Cache2 *clustering.ClusterCache[string, string]
}

func setupTwoNodeCluster(t *testing.T) twoNodeCluster {
	t.Helper()
	clk := clock.New()

	// --- Node 1 ---
	mux1 := cluster.NewMessageMux()
	c1, err := cluster.New(cluster.Config{
		Region:    "us-east-1",
		NodeID:    "node-1",
		BindAddr:  "127.0.0.1",
		OnMessage: mux1.OnMessage,
	})
	require.NoError(t, err)
	b1 := clustering.NewGossipBroadcaster(c1)
	cluster.Subscribe(mux1, b1.HandleCacheInvalidation)

	d1, err := clustering.NewInvalidationDispatcher(b1)
	require.NoError(t, err)

	lc1, err := cache.New(cache.Config[string, string]{
		Fresh: time.Minute, Stale: time.Hour, MaxSize: 1000,
		Resource: "test_cache", Clock: clk,
	})
	require.NoError(t, err)

	cc1, err := clustering.New(clustering.Config[string, string]{
		LocalCache: lc1, Broadcaster: b1, Dispatcher: d1, NodeID: "node-1",
	})
	require.NoError(t, err)

	// --- Node 2 ---
	mux2 := cluster.NewMessageMux()
	c1Addr := c1.Members()[0].FullAddress().Addr
	time.Sleep(50 * time.Millisecond)

	c2, err := cluster.New(cluster.Config{
		Region:    "us-east-1",
		NodeID:    "node-2",
		BindAddr:  "127.0.0.1",
		LANSeeds:  []string{c1Addr},
		OnMessage: mux2.OnMessage,
	})
	require.NoError(t, err)
	b2 := clustering.NewGossipBroadcaster(c2)
	cluster.Subscribe(mux2, b2.HandleCacheInvalidation)

	d2, err := clustering.NewInvalidationDispatcher(b2)
	require.NoError(t, err)

	lc2, err := cache.New(cache.Config[string, string]{
		Fresh: time.Minute, Stale: time.Hour, MaxSize: 1000,
		Resource: "test_cache", Clock: clk,
	})
	require.NoError(t, err)

	cc2, err := clustering.New(clustering.Config[string, string]{
		LocalCache: lc2, Broadcaster: b2, Dispatcher: d2, NodeID: "node-2",
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, cc1.Close())
		require.NoError(t, cc2.Close())
		require.NoError(t, c2.Close())
		require.NoError(t, c1.Close())
	})

	// Wait for cluster to form
	require.Eventually(t, func() bool {
		return len(c1.Members()) == 2 && len(c2.Members()) == 2
	}, 5*time.Second, 100*time.Millisecond, "nodes should discover each other")

	return twoNodeCluster{Cache1: cc1, Cache2: cc2}
}

func TestGossipCacheInvalidation_Remove(t *testing.T) {
	ctx := context.Background()
	tc := setupTwoNodeCluster(t)

	// Set a value on node 2
	tc.Cache2.Set(ctx, "test-key", "test-value")
	val, hit := tc.Cache2.Get(ctx, "test-key")
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "test-value", val)

	// Remove on node 1 — should propagate to node 2
	tc.Cache1.Remove(ctx, "test-key")

	require.Eventually(t, func() bool {
		_, hit := tc.Cache2.Get(ctx, "test-key")
		return hit == cache.Miss
	}, 5*time.Second, 100*time.Millisecond, "key should be invalidated on node 2")
}

func TestGossipCacheInvalidation_Clear(t *testing.T) {
	ctx := context.Background()
	tc := setupTwoNodeCluster(t)

	// Populate node 2's cache with multiple keys
	tc.Cache2.Set(ctx, "key-a", "value-a")
	tc.Cache2.Set(ctx, "key-b", "value-b")
	tc.Cache2.Set(ctx, "key-c", "value-c")

	_, hit := tc.Cache2.Get(ctx, "key-a")
	require.Equal(t, cache.Hit, hit)

	// Clear on node 1 — should propagate and clear node 2's cache
	tc.Cache1.Clear(ctx)

	require.Eventually(t, func() bool {
		_, hitA := tc.Cache2.Get(ctx, "key-a")
		_, hitB := tc.Cache2.Get(ctx, "key-b")
		_, hitC := tc.Cache2.Get(ctx, "key-c")
		return hitA == cache.Miss && hitB == cache.Miss && hitC == cache.Miss
	}, 5*time.Second, 100*time.Millisecond, "all keys should be cleared on node 2")
}
