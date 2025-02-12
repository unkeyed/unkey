package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	connect "connectrpc.com/connect"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	connectSrv "github.com/unkeyed/unkey/apps/agent/pkg/connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestSync(t *testing.T) {
	type Node struct {
		srv     *service
		cluster cluster.Cluster
	}

	nodes := []Node{}
	logger := logging.New(nil)
	serfAddrs := []string{}

	clusterSize := 3
	for i := range clusterSize {
		node := Node{}
		c, serfAddr, rpcAddr := createCluster(t, fmt.Sprintf("node-%d", i), serfAddrs)
		serfAddrs = append(serfAddrs, serfAddr)
		node.cluster = c

		srv, err := New(Config{
			Logger:  logger,
			Metrics: metrics.NewNoop(),
			Cluster: c,
		})
		require.NoError(t, err)
		node.srv = srv
		nodes = append(nodes, node)

		cSrv, err := connectSrv.New(connectSrv.Config{
			Logger:  logger,
			Metrics: metrics.NewNoop(),
			Image:   "does not matter",
		})
		require.NoError(t, err)
		err = cSrv.AddService(connectSrv.NewRatelimitServer(srv, logger, "test-auth-token"))
		require.NoError(t, err)

		require.NoError(t, err)
		go func() {
			t.Logf("rpcAddr: %s", rpcAddr)
			u, err := url.Parse(rpcAddr)
			require.NoError(t, err)

			err = cSrv.Listen(u.Host)
			require.NoError(t, err)

		}()

		require.Eventually(t, func() bool {
			client := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, rpcAddr)
			res, livenessErr := client.Liveness(context.Background(), connect.NewRequest(&ratelimitv1.LivenessRequest{}))
			require.NoError(t, livenessErr)
			return res.Msg.Status == "ok"

		},
			time.Minute, 100*time.Millisecond)
	}
	require.Len(t, nodes, clusterSize)
	require.Len(t, serfAddrs, clusterSize)

	for _, n := range nodes {
		require.Eventually(t, func() bool {
			return n.cluster.Size() == clusterSize
		}, time.Minute, 100*time.Millisecond)
	}

	ctx := context.Background()

	identifier := uid.New("test")
	limit := int64(10)
	duration := time.Minute

	now := time.Now()
	key := bucketKey{identifier, limit, duration}
	req := &ratelimitv1.RatelimitRequest{
		Time:       util.Pointer(now.UnixMilli()),
		Identifier: identifier,
		Limit:      limit,
		Duration:   duration.Milliseconds(),
		Cost:       1,
	}

	// Figure out who is the origin
	_, err := nodes[0].srv.Ratelimit(ctx, req)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	originIndex := 0
	for i, n := range nodes {
		n.srv.bucketsMu.RLock()
		buckets := len(n.srv.buckets)
		n.srv.bucketsMu.RUnlock()
		t.Logf("node %d: found %d buckets", i, buckets)
		if buckets > 0 {
			originIndex = i
		}
	}
	t.Logf("originIndex: %d", originIndex)

	// Run ratelimit once against each node
	for i, n := range nodes {
		res, err := n.srv.Ratelimit(context.Background(), req)
		require.NoError(t, err)
		require.True(t, res.Success)
		t.Logf("1st pass: %d: %+v", i, res)
	}

	require.Eventually(t, func() bool {
		nodes[originIndex].srv.bucketsMu.RLock()
		bucket, ok := nodes[originIndex].srv.getBucket(key)
		nodes[originIndex].srv.bucketsMu.RUnlock()
		require.True(t, ok)
		bucket.RLock()
		window := bucket.getCurrentWindow(now)
		counter := window.Counter
		bucket.RUnlock()

		return counter == int64(len(nodes)+1)
	}, 10*time.Second, time.Second)

	for _, n := range nodes {
		require.NoError(t, n.cluster.Shutdown())
	}
}

func createCluster(
	t *testing.T,
	nodeId string,
	joinAddrs []string,
) (c cluster.Cluster, serfAddr string, rpcAddr string) {
	t.Helper()

	logger := logging.New(nil).With().Str("nodeId", nodeId).Logger().Level(zerolog.ErrorLevel)

	p := port.New()

	rpcAddr = fmt.Sprintf("http://localhost:%d", p.Get())
	serfAddr = fmt.Sprintf("localhost:%d", p.Get())
	m, err := membership.New(membership.Config{
		NodeId:   nodeId,
		Logger:   logger,
		SerfAddr: serfAddr,
		RpcAddr:  rpcAddr,
	})
	require.NoError(t, err)

	_, err = m.Join(joinAddrs...)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		peers, membersErr := m.Members()
		require.NoError(t, membersErr)
		return len(peers) == len(joinAddrs)+1
	}, time.Minute, 100*time.Millisecond)

	c, err = cluster.New(cluster.Config{
		NodeId:     nodeId,
		Membership: m,
		Logger:     logger,
		Metrics:    metrics.NewNoop(),
		Debug:      true,
		RpcAddr:    rpcAddr,
		AuthToken:  "test-auth-token",
	})
	require.NoError(t, err)

	return c, serfAddr, rpcAddr

}

// DO NOT USE OUTSIDE OF TESTS
// Return the value of a map where the key is the largest
func getLatestWindow[K int64, V any](m map[K]V) V {

	var max K
	var window V
	for k, v := range m {
		if k >= max {
			max = k
			window = v

		}
	}
	return window

}
