package analytics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestGetGatewayRequests(t *testing.T) {
	args := `analytics get-gateway-requests --query='SELECT * FROM gateway_requests_v1'`
	want := components.V2AnalyticsGetGatewayRequestsRequestBody{
		Query: "SELECT * FROM gateway_requests_v1",
	}

	got := testutil.CaptureRequestWithData[components.V2AnalyticsGetGatewayRequestsRequestBody](t, Cmd(), args, []any{})
	require.Equal(t, want, got)
}
