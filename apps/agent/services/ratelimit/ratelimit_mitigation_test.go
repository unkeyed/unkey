package ratelimit_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	connect "connectrpc.com/connect"

	"github.com/stretchr/testify/require"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	connectSrv "github.com/unkeyed/unkey/apps/agent/pkg/connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
)

func TestExceedingTheLimitShouldNotifyAllNodes(t *testing.T) {
	t.Skip()
	for _, clusterSize := range []int{1, 3, 5} {
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
				u, err := url.Parse(rpcAddr)
				require.NoError(t, err)
				go srv.Listen(u.Host)

				require.Eventually(t,
					func() bool {
						client := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, rpcAddr)
						res, livenessErr := client.Liveness(context.Background(), connect.NewRequest(&ratelimitv1.LivenessRequest{}))
						require.NoError(t, livenessErr)
						return res.Msg.Status == "ok"

					},
					time.Minute,
					100*time.Millisecond)

			}
			require.Len(t, ratelimiters, clusterSize)
			require.Len(t, serfAddrs, clusterSize)

			for _, c := range clusters {
				require.Eventually(t, func() bool {
					return c.Size() == clusterSize
				}, time.Minute, 100*time.Millisecond)
			}

			identifier := uid.New("test")
			limit := int64(10)
			duration := time.Minute

			now := time.Now()
			req := &ratelimitv1.RatelimitRequest{
				Time:       util.Pointer(now.UnixMilli()),
				Identifier: identifier,
				Limit:      limit,
				Duration:   duration.Milliseconds(),
				Cost:       1,
			}
			ctx := context.Background()

			// Saturate the window
			for i := int64(0); i <= limit; i++ {
				rl := util.RandomElement(ratelimiters)
				res, err := rl.Ratelimit(ctx, req)
				require.NoError(t, err)
				t.Logf("saturate res: %+v", res)
				require.True(t, res.Success)
			}

			time.Sleep(time.Second * 5)

			// Let's hit everry node again
			// They should all be mitigated
			for i, rl := range ratelimiters {
				res, err := rl.Ratelimit(ctx, req)
				require.NoError(t, err)
				t.Logf("res from %d: %+v", i, res)
				// require.False(t, res.Success)
			}

		})

	}
}
