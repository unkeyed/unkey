package handler_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1RatelimitRatelimit "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_ratelimit_ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/testutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestRatelimit(t *testing.T) {
	h := testutil.NewHarness(t)
	h.Register(v1RatelimitRatelimit.Register)

	req := v1RatelimitRatelimit.V1RatelimitRatelimitRequest{}
	req.Body.Identifier = uid.New("test")
	req.Body.Limit = 10
	req.Body.Duration = 1000

	resp := h.Api().Post("/ratelimit.v1.RatelimitService/Ratelimit", req.Body)

	respBody := v1RatelimitRatelimit.V1RatelimitRatelimitResponse{}
	err := json.Unmarshal(resp.Body.Bytes(), &respBody.Body)
	require.NoError(t, err)

	require.Equal(t, 200, resp.Code)
	require.Equal(t, int64(10), respBody.Body.Limit)
	require.Equal(t, int64(9), respBody.Body.Remaining)
	require.Equal(t, true, respBody.Body.Success)
	require.Equal(t, int64(1), respBody.Body.Current)
}

func TestRatelimitWithLease(t *testing.T) {

	h := testutil.NewHarness(t)

	h.Register(v1RatelimitRatelimit.Register)

	req := v1RatelimitRatelimit.V1RatelimitRatelimitRequest{}
	req.Body.Identifier = uid.New("test")
	req.Body.Limit = 100
	req.Body.Duration = time.Minute.Milliseconds()
	req.Body.Lease = &v1RatelimitRatelimit.Lease{
		Cost:    10,
		Timeout: 10 * time.Second.Milliseconds(),
	}

	resp := h.Api().Post("/ratelimit.v1.RatelimitService/Ratelimit", req.Body)

	respBody := v1RatelimitRatelimit.V1RatelimitRatelimitResponse{}
	err := json.Unmarshal(resp.Body.Bytes(), &respBody.Body)
	require.NoError(t, err)

	require.Equal(t, 200, resp.Code)
	require.Equal(t, int64(100), respBody.Body.Limit)
	require.Equal(t, int64(90), respBody.Body.Remaining)
	require.Equal(t, true, respBody.Body.Success)
	require.Equal(t, int64(10), respBody.Body.Current)
	require.NotNil(t, respBody.Body.Lease)
}
