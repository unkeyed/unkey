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
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
)

var CLUSTER_SIZES = []int{1, 3, 20}

func TestRatelimit_Consistency(t *testing.T) {

	testCases := []struct {
		name        string
		limit       int64
		duration    int64
		rps         int
		seconds     int
		expectedMin int
		expectedMax int
	}{
		{
			name:        "Basic Test",
			limit:       100,
			duration:    10000,
			rps:         10,
			seconds:     20,
			expectedMin: 200,
			expectedMax: 200,
		},
		{
			name:        "High Rate with Short Window",
			limit:       500,
			duration:    5000,
			rps:         100,
			seconds:     10,
			expectedMin: 900,
			expectedMax: 1000,
		},
		{
			name:        "Constant Rate Equals Limit",
			limit:       200,
			duration:    10000,
			rps:         20,
			seconds:     20,
			expectedMin: 400,
			expectedMax: 400,
		},
		{
			name:        "Rate Lower Than Limit",
			limit:       500,
			duration:    10000,
			rps:         100,
			seconds:     10,
			expectedMin: 500,
			expectedMax: 1000,
		},
		{
			name:        "Rate Higher Than Limit",
			limit:       100,
			duration:    5000,
			rps:         200,
			seconds:     15,
			expectedMin: 300,
			expectedMax: 1500,
		},
		{
			name:        "High Burst",
			limit:       10,
			duration:    1000,
			rps:         2000,
			seconds:     15,
			expectedMin: 150,
			expectedMax: 500,
		},
		{
			name:        "Very Long Window",
			limit:       100,
			duration:    120000,
			rps:         1,
			seconds:     10,
			expectedMin: 10,
			expectedMax: 10,
		},
	}

	for _, clusterSize := range CLUSTER_SIZES {
		t.Run(fmt.Sprintf("Cluster Size %d", clusterSize), func(t *testing.T) {
			t.Parallel()
			for _, tc := range testCases {
				t.Run(fmt.Sprintf(
					"%s, [~%ds], passed requests are within [%d - %d]",
					tc.name,
					tc.seconds,
					tc.expectedMin,
					tc.expectedMax,
				), func(t *testing.T) {
					logger := logging.New(nil).Level(zerolog.ErrorLevel)
					clusters := []cluster.Cluster{}
					ratelimiters := []ratelimit.Service{}
					serfAddrs := []string{}

					for i := range clusterSize {
						c, serfAddr, rpcAddr := createCluster(t, fmt.Sprintf("node-%d", i), serfAddrs)
						serfAddrs = append(serfAddrs, serfAddr)
						clusters = append(clusters, c)

						rl, err := ratelimit.New(ratelimit.Config{
							Logger:  logger,
							Cluster: c,
						})
						require.NoError(t, err)
						ratelimiters = append(ratelimiters, rl)

						srv, err := connectSrv.New(connectSrv.Config{
							Logger: logger,
							Image:  "does not matter",
						})
						require.NoError(t, err)
						err = srv.AddService(connectSrv.NewRatelimitServer(rl, logger, "test-auth-token"))
						require.NoError(t, err)

						go func() {
							_ = srv.Listen(rpcAddr)
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

					// Pick up to 3 nodes for ingress requests
					// Right now our cluster converges too slowly to accept ratelimits from all nodes
					max := 3
					if len(ratelimiters) < max {
						max = len(ratelimiters)
					}
					ingressNodes := ratelimiters[:max]
					require.Len(t, ingressNodes, max)

					results := loadTest(t, tc.rps, tc.seconds, func() *ratelimitv1.RatelimitResponse {

						rl := util.RandomElement(ingressNodes)
						res, err := rl.Ratelimit(context.Background(), &ratelimitv1.RatelimitRequest{
							Identifier: "test",
							Limit:      tc.limit,
							Duration:   tc.duration,
							Cost:       1,
						})
						require.NoError(t, err)
						return res
					})

					require.Len(t, results, tc.rps*tc.seconds)

					passed := 0
					for _, res := range results {
						if res.Success {
							passed++
						}
					}

					require.GreaterOrEqual(t, passed, tc.expectedMin)
					require.LessOrEqual(t, passed, tc.expectedMax)

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
		Debug:      true,
		RpcAddr:    rpcAddr,
		AuthToken:  "test-auth-token",
	})
	require.NoError(t, err)

	return c, serfAddr, rpcAddr

}

func loadTest[T any](t *testing.T, rps int, seconds int, fn func() T) []T {
	t.Helper()

	resultsC := make(chan T)

	var wg sync.WaitGroup

	for range seconds {
		for range rps {
			wg.Add(1)
			go func() {
				res := fn()
				resultsC <- res
				wg.Done()
			}()
		}
		time.Sleep(time.Second)
	}

	results := []T{}
	go func() {
		for res := range resultsC {
			results = append(results, res)
		}
	}()
	wg.Wait()

	return results

}
