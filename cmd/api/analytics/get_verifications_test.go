package analytics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetVerifications(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2AnalyticsGetVerificationsRequestBody
	}{
		{
			name: "simple query",
			args: `analytics get-verifications --query=SELECT(1)`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT(1)",
			},
		},
		{
			name: "count query",
			args: `analytics get-verifications --query='SELECT COUNT(*) FROM verifications'`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT COUNT(*) FROM verifications",
			},
		},
		{
			name: "query with filter",
			args: `analytics get-verifications --query="SELECT key_id, outcome FROM key_verifications_v1 WHERE outcome = 'VALID'"`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT key_id, outcome FROM key_verifications_v1 WHERE outcome = 'VALID'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[openapi.V2AnalyticsGetVerificationsRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
