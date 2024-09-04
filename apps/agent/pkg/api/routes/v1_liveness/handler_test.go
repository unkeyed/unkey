package v1Liveness_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	v1Liveness "github.com/unkeyed/unkey/apps/agent/pkg/api/routes/v1_liveness"
	"github.com/unkeyed/unkey/apps/agent/pkg/api/testutil"
)

func TestLiveness(t *testing.T) {

	h := testutil.NewHarness(t)
	route := h.SetupRoute(v1Liveness.New)
	res := testutil.CallRoute[any, openapi.V1LivenessResponseBody](t, route, nil, nil)

	require.Equal(t, 200, res.Status)
}
