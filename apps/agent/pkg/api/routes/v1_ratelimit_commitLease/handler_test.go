package v1RatelimitCommitLease_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1RatelimitCommitLease "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_commitLease"
	v1RatelimitRatelimit "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestCommitLease(t *testing.T) {
	t.Skip()
	h := testutil.NewHarness(t)
	ratelimitRoute := h.SetupRoute(v1RatelimitRatelimit.New)
	commitLeaseRoute := h.SetupRoute(v1RatelimitCommitLease.New)

	req := openapi.V1RatelimitRatelimitRequestBody{

		Identifier: uid.New("test"),
		Limit:      100,
		Duration:   time.Minute.Milliseconds(),
		Cost:       util.Pointer[int64](0),
		Lease: &openapi.Lease{
			Cost:    10,
			Timeout: 10 * time.Second.Milliseconds(),
		},
	}

	res := testutil.CallRoute[openapi.V1RatelimitRatelimitRequestBody, openapi.V1RatelimitRatelimitResponseBody](t, ratelimitRoute, nil, req)

	require.Equal(t, 200, res.Status)
	require.Equal(t, int64(100), res.Body.Limit)
	require.Equal(t, int64(90), res.Body.Remaining)
	require.Equal(t, true, res.Body.Success)
	require.Equal(t, int64(10), res.Body.Current)
	require.NotNil(t, res.Body.Lease)

	commitReq := openapi.V1RatelimitCommitLeaseRequestBody{
		Cost:  5,
		Lease: res.Body.Lease,
	}

	commitRes := testutil.CallRoute[openapi.V1RatelimitCommitLeaseRequestBody, any](t, commitLeaseRoute, nil, commitReq)

	require.Equal(t, 204, commitRes.Status)

}
