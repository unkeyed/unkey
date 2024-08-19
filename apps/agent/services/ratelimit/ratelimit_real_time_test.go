package ratelimit_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
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

func TestRatelimit_Accuracy(t *testing.T) {

	testCases := []struct {
		limit    int64
		duration int64
		rps      int64
		seconds  int64
	}{
		// below the limit
		{
			limit:    100,
			duration: 10_000,
			rps:      10,
			seconds:  60,
		},
		{
			limit:    200,
			duration: 10_000,
			rps:      100,
			seconds:  60,
		},
		{
			limit:    10,
			duration: 10000,
			rps:      15,
			seconds:  120,
		},
		{
			limit:    20,
			duration: 1000,
			rps:      50,
			seconds:  60,
		},
		{
			limit:    200,
			duration: 10000,
			rps:      20,
			seconds:  20,
		},
		{
			limit:    500,
			duration: 10000,
			rps:      100,
			seconds:  30,
		},
		{
			limit:    100,
			duration: 5000,
			rps:      200,
			seconds:  120,
		},
	}

	for _, clusterSize := range []int{9} {
		t.Run(fmt.Sprintf("Cluster Size %d", clusterSize), func(t *testing.T) {
			logger := logging.New(nil)
			clusters := []cluster.Cluster{}
			ratelimiters := []ratelimit.Service{}
			serfAddrs := []string{}

			for i := range clusterSize {
				c, serfAddr, rpcAddr := createCluster(t, fmt.Sprintf("node-%d", i), serfAddrs)
				serfAddrs = append(serfAddrs, serfAddr)
				clusters = append(clusters, c)

				rl, err := ratelimit.New(ratelimit.Config{
					Logger:  logger,
					Metrics: metrics.NewNoop(),
					Cluster: c,
				})
				require.NoError(t, err)
				ratelimiters = append(ratelimiters, rl)

				srv, err := connectSrv.New(connectSrv.Config{
					Logger:  logger,
					Metrics: metrics.NewNoop(),
					Image:   "does not matter",
				})
				require.NoError(t, err)
				err = srv.AddService(connectSrv.NewRatelimitServer(rl, logger, "test-auth-token"))
				require.NoError(t, err)

				require.NoError(t, err)
				go func() {
					err := srv.Listen(rpcAddr)
					require.NoError(t, err)

				}()

				require.Eventually(t, func() bool {
					client := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, fmt.Sprintf("http://%s", rpcAddr))
					res, livenessErr := client.Liveness(context.Background(), connect.NewRequest(&ratelimitv1.LivenessRequest{}))
					require.NoError(t, livenessErr)
					return res.Msg.Status == "ok"

				},
					time.Minute, 100*time.Millisecond)

			}
			require.Len(t, ratelimiters, clusterSize)
			require.Len(t, serfAddrs, clusterSize)

			for _, c := range clusters {
				require.Eventually(t, func() bool {
					return c.Size() == clusterSize
				}, time.Minute, 100*time.Millisecond)
			}
			for _, tc := range testCases {
				t.Run(fmt.Sprintf("[%d / %ds], attacked with %d rps for %ds",
					tc.limit,
					tc.duration/1000,
					tc.rps,
					tc.seconds,
				), func(t *testing.T) {

					for _, nIngressNodes := range []int{1, 3, 5, clusterSize} {
						if nIngressNodes > clusterSize {
							nIngressNodes = clusterSize
						}
						t.Run(fmt.Sprintf("%d ingress nodes", nIngressNodes), func(t *testing.T) {

							identifier := uid.New("test")

							// Pick a few nodes for ingress requests
							ingressNodes := ratelimiters[:nIngressNodes]

							results := loadTest(t, tc.rps, tc.seconds, func() *ratelimitv1.RatelimitResponse {

								rl := util.RandomElement(ingressNodes)
								res, err := rl.Ratelimit(context.Background(), &ratelimitv1.RatelimitRequest{
									Identifier: identifier,
									Limit:      tc.limit,
									Duration:   tc.duration,
									Cost:       1,
								})
								require.NoError(t, err)
								return res
							})

							require.Len(t, results, int(tc.rps*tc.seconds))

							passed := int64(0)
							for _, res := range results {
								require.GreaterOrEqual(t, res.Remaining, int64(0))
								require.GreaterOrEqual(t, res.Current, int64(0))
								if res.Success {
									passed++
								}
							}

							// At least 95% of the requests should pass
							lower := 0.95
							// At most 150% + 75% per additional ingress node should pass
							upper := 1.50 + 0.80*float64(len(ingressNodes)-1)

							exactLimit := (tc.limit / (tc.duration / 1000)) * tc.seconds
							require.GreaterOrEqual(t, passed, int64(float64(exactLimit)*lower))
							require.LessOrEqual(t, passed, int64(float64(exactLimit)*upper))
						})

					}
				})
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

	rpcAddr = fmt.Sprintf("localhost:%d", port.Get())
	serfAddr = fmt.Sprintf("localhost:%d", port.Get())
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

func loadTest[T any](t *testing.T, rps int64, seconds int64, fn func() T) []T {
	t.Helper()

	resultsC := make(chan T)

	var wg sync.WaitGroup

	for range seconds {
		for range rps {
			time.Sleep(time.Second / time.Duration(rps))

			wg.Add(1)
			go func() {
				resultsC <- fn()
			}()
		}
	}

	results := []T{}
	go func() {
		for res := range resultsC {
			results = append(results, res)
			wg.Done()

		}
	}()
	wg.Wait()

	return results

}
