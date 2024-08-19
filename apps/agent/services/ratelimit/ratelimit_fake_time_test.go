package ratelimit_test

import (
	"context"
	"fmt"
	"net/http"
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

func TestAccuracy_fixed_time(t *testing.T) {
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

			for _, limit := range []int64{
				5,
				10,
				100,
				500,
			} {
				for _, duration := range []time.Duration{
					1 * time.Second,
					10 * time.Second,
					1 * time.Minute,
					5 * time.Minute,
					1 * time.Hour,
				} {
					for _, windows := range []int64{1, 2, 5, 10, 50} {
						requests := limit * windows * 100
						t.Run(fmt.Sprintf("rate %d/%s  %d requests across %d windows",
							limit,
							duration,
							requests,
							windows,
						), func(t *testing.T) {

							for _, nIngressNodes := range []int{1, 3, 5, clusterSize} {
								if nIngressNodes > clusterSize {
									nIngressNodes = clusterSize
								}
								t.Run(fmt.Sprintf("%d ingress nodes", nIngressNodes), func(t *testing.T) {

									identifier := uid.New("test")
									ingressNodes := ratelimiters[:nIngressNodes]

									now := time.Now()
									end := now.Add(duration * time.Duration(windows))
									passed := int64(0)

									dt := duration * time.Duration(windows) / time.Duration(requests)

									for i := now; i.Before(end); i = i.Add(dt) {
										rl := util.RandomElement(ingressNodes)

										res, err := rl.Ratelimit(context.Background(), &ratelimitv1.RatelimitRequest{
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

									// At least 95% of the requests should pass
									// lower := 0.95
									// At most 150% + 75% per additional ingress node should pass
									upper := 1.50 + 1.0*float64(len(ingressNodes)-1)

									exactLimit := limit * (windows + 1)
									// require.GreaterOrEqual(t, passed, int64(float64(exactLimit)*lower))
									require.LessOrEqual(t, passed, int64(float64(exactLimit)*upper))

								})
							}

						})
					}

				}
			}
		})
	}
}
