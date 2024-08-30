package handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1RatelimitCommitLease "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_commitLease"
	v1RatelimitRatelimit "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

func TestCommitLease(t *testing.T) {
	t.Skip()
	h := testutil.NewHarness(t)

	h.Register(v1RatelimitRatelimit.Register)
	h.Register(v1RatelimitCommitLease.Register)

	ratelimitReq := v1RatelimitRatelimit.V1RatelimitRatelimitRequest{}
	ratelimitReq.Body.Identifier = uid.New("test")
	ratelimitReq.Body.Limit = 100
	ratelimitReq.Body.Duration = time.Minute.Milliseconds()
	ratelimitReq.Body.Cost = util.Pointer[int64](0)
	ratelimitReq.Body.Lease = &v1RatelimitRatelimit.Lease{
		Cost:    10,
		Timeout: 10 * time.Second.Milliseconds(),
	}

	res := h.Api().Post("/ratelimit.v1.RatelimitService/Ratelimit", ratelimitReq.Body)

	responseBody := v1RatelimitRatelimit.V1RatelimitRatelimitResponse{}
	testutil.UnmarshalBody(t, res, &responseBody.Body)

	t.Logf("responseBody: %+v", responseBody)
	require.Equal(t, 200, res.Code)
	require.Equal(t, int64(100), responseBody.Body.Limit)
	require.Equal(t, int64(90), responseBody.Body.Remaining)
	require.Equal(t, true, responseBody.Body.Success)
	require.Equal(t, int64(10), responseBody.Body.Current)
	require.NotNil(t, responseBody.Body.Lease)

	commitReq := v1RatelimitCommitLease.V1RatelimitCommitLeaseRequest{}
	commitReq.Body.Cost = 5
	commitReq.Body.Lease = *responseBody.Body.Lease

	commitRes := h.Api().Post("/v1/ratelimit.commitLease", commitReq.Body)

	require.Equal(t, 204, commitRes.Code)

}
