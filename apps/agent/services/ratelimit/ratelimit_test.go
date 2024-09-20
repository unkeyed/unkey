package ratelimit_test

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
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
)

func TestAccuracy_fixed_time(t *testing.T) {

	for _, clusterSize := range []int{1, 3, 5} {
		t.Run(fmt.Sprintf("ClusterSize:%d", clusterSize), func(t *testing.T) {
			for _, limit := range []int64{
				5,
				10,
				100,
			} {
				type Node struct {
					srv     ratelimit.Service
					cluster cluster.Cluster
				}

				nodes := []Node{}
				logger := logging.New(nil)
				serfAddrs := []string{}

				for i := 0; i < clusterSize; i++ {
					node := Node{}
					c, serfAddr, rpcAddr := createCluster(t, fmt.Sprintf("node-%d", i), serfAddrs)
					serfAddrs = append(serfAddrs, serfAddr)
					node.cluster = c

					srv, err := ratelimit.New(ratelimit.Config{
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
					u, err := url.Parse(rpcAddr)
					require.NoError(t, err)

					go cSrv.Listen(u.Host)

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

				t.Run(fmt.Sprintf("limit:%d", limit), func(t *testing.T) {

					for _, duration := range []time.Duration{
						1 * time.Second,
						10 * time.Second,
						1 * time.Minute,
						5 * time.Minute,
						1 * time.Hour,
					} {
						t.Run(fmt.Sprintf("duration:%s", duration), func(t *testing.T) {

							for _, windows := range []int64{1, 2, 5, 10, 50} {
								t.Run(fmt.Sprintf("windows:%d", windows), func(t *testing.T) {

									// Attack the ratelimit with 100x as much as it should let pass
									requests := limit * windows * 100

									for _, nIngressNodes := range []int{1, 3, clusterSize} {
										if nIngressNodes > clusterSize {
											nIngressNodes = clusterSize
										}
										t.Run(fmt.Sprintf("%d/%d ingress nodes",
											nIngressNodes,
											clusterSize,
										), func(t *testing.T) {

											identifier := uid.New("test")
											ingressNodes := nodes[:nIngressNodes]

											now := time.Now()
											end := now.Add(duration * time.Duration(windows))
											passed := int64(0)

											dt := duration * time.Duration(windows) / time.Duration(requests)

											for i := now; i.Before(end); i = i.Add(dt) {
												rl := util.RandomElement(ingressNodes)

												res, err := rl.srv.Ratelimit(context.Background(), &ratelimitv1.RatelimitRequest{
													// random time within one of the windows
													Time:       util.Pointer(i.UnixMilli()),
													Identifier: identifier,
													Limit:      limit,
													Duration:   duration.Milliseconds(),
													Cost:       1,
												})
												require.NoError(t, err)
												if res.Success {
													passed++
												}
											}

											exactLimit := limit * (windows + 1)
											lower := exactLimit
											// At most 150% + 75% per additional ingress node should pass
											upper := 1.50 + 1.0*float64(len(ingressNodes)-1)

											require.GreaterOrEqual(t, passed, lower)
											require.LessOrEqual(t, passed, int64(float64(exactLimit)*upper))

										})
									}

								})
							}
						})
					}
				})

				for _, n := range nodes {
					require.NoError(t, n.cluster.Shutdown())
				}

			}

		})
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
