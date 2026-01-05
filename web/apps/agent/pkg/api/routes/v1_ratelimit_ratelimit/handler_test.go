package v1RatelimitRatelimit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1RatelimitRatelimit "github.com/unkeyed/unkey/svc/agent/pkg/api/routes/v1_ratelimit_ratelimit"
	"github.com/unkeyed/unkey/svc/agent/pkg/api/testutil"
	"github.com/unkeyed/unkey/svc/agent/pkg/openapi"
	"github.com/unkeyed/unkey/svc/agent/pkg/uid"
)

func TestRatelimit(t *testing.T) {
	h := testutil.NewHarness(t)
	route := h.SetupRoute(v1RatelimitRatelimit.New)

	req := openapi.V1RatelimitRatelimitRequestBody{
		Identifier: uid.New("test"),
		Limit:      10,
		Duration:   1000,
	}

	resp := testutil.CallRoute[openapi.V1RatelimitRatelimitRequestBody, openapi.V1RatelimitRatelimitResponseBody](t, route, nil, req)

	require.Equal(t, 200, resp.Status)
	require.Equal(t, int64(10), resp.Body.Limit)
	require.Equal(t, int64(9), resp.Body.Remaining)
	require.Equal(t, true, resp.Body.Success)
	require.Equal(t, int64(1), resp.Body.Current)
}

func TestRatelimitWithLease(t *testing.T) {
	t.Skip()
	h := testutil.NewHarness(t)
	route := h.SetupRoute(v1RatelimitRatelimit.New)

	req := openapi.V1RatelimitRatelimitRequestBody{

		Identifier: uid.New("test"),
		Limit:      100,
		Duration:   time.Minute.Milliseconds(),
		Lease: &openapi.Lease{
			Cost:    10,
			Timeout: 10 * time.Second.Milliseconds(),
		},
	}
	resp := testutil.CallRoute[openapi.V1RatelimitRatelimitRequestBody, openapi.V1RatelimitRatelimitResponseBody](t, route, nil, req)

	require.Equal(t, 200, resp.Status)
	require.Equal(t, int64(100), resp.Body.Limit)
	require.Equal(t, int64(90), resp.Body.Remaining)
	require.Equal(t, true, resp.Body.Success)
	require.Equal(t, int64(10), resp.Body.Current)
	require.NotNil(t, resp.Body.Lease)
}
