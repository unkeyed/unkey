package handler_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/routes"
	v1RatelimitRatelimit "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/membership"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/port"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
)

func TestRatelimit(t *testing.T) {
	_, api := humatest.New(t)

	nodeId := uid.New("test")
	authToken := uid.New("test")
	serfAddr := fmt.Sprintf("localhost:%d", port.Get())
	rpcAddr := fmt.Sprintf("localhost:%d", port.Get())

	l := logging.NewNoopLogger()
	m := metrics.NewNoop()

	memb, err := membership.New(membership.Config{
		NodeId:   nodeId,
		SerfAddr: serfAddr,
	})
	require.NoError(t, err)

	c, err := cluster.New(cluster.Config{
		NodeId:     nodeId,
		Membership: memb,
		Logger:     l,
		Metrics:    m,
		AuthToken:  authToken,
		RpcAddr:    rpcAddr,
	})
	require.NoError(t, err)
	rl, err := ratelimit.New(ratelimit.Config{
		Logger:  l,
		Metrics: m,
		Cluster: c,
	})
	require.NoError(t, err)

	v1RatelimitRatelimit.Register(api, routes.Services{
		Ratelimit: rl,
	})

	req := v1RatelimitRatelimit.V1RatelimitRatelimitRequest{}
	req.Body.Identifier = uid.New("test")
	req.Body.Limit = 10
	req.Body.Duration = 1000

	resp := api.Post("/ratelimit.v1.RatelimitService/Ratelimit", req.Body)

	respBody := v1RatelimitRatelimit.V1RatelimitRatelimitResponse{}
	err = json.Unmarshal(resp.Body.Bytes(), &respBody.Body)
	require.NoError(t, err)

	require.Equal(t, 200, resp.Code)
	require.Equal(t, int64(10), respBody.Body.Limit)
	require.Equal(t, int64(9), respBody.Body.Remaining)
	require.Equal(t, true, respBody.Body.Success)
	require.Equal(t, int64(1), respBody.Body.Current)
}
