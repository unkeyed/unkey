package analytics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetRatelimits(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2AnalyticsGetRatelimitsRequestBody
	}{
		{
			name: "raw data query",
			args: `analytics get-ratelimits --query=SELECT+*+FROM+ratelimits_v1`,
			want: openapi.V2AnalyticsGetRatelimitsRequestBody{
				Query: "SELECT+*+FROM+ratelimits_v1",
			},
		},
		{
			name: "aggregate query",
			args: `analytics get-ratelimits --query=SELECT+namespace_id,COUNT(*)+FROM+ratelimits_per_hour_v1+GROUP+BY+namespace_id`,
			want: openapi.V2AnalyticsGetRatelimitsRequestBody{
				Query: "SELECT+namespace_id,COUNT(*)+FROM+ratelimits_per_hour_v1+GROUP+BY+namespace_id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[openapi.V2AnalyticsGetRatelimitsRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
