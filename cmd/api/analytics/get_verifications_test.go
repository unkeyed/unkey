package analytics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetVerifications(t *testing.T) {
	// Note: the shared test harness splits args with strings.Fields, so query
	// values must not contain spaces.  This is fine because we are testing
	// flag-to-request mapping, not SQL validity.
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
			name: "count query without spaces",
			args: `analytics get-verifications --query=SELECT+COUNT(*)+FROM+verifications`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT+COUNT(*)+FROM+verifications",
			},
		},
		{
			name: "query with filter",
			args: `analytics get-verifications --query=SELECT+key_id,outcome+FROM+key_verifications_v1+WHERE+outcome='VALID'`,
			want: openapi.V2AnalyticsGetVerificationsRequestBody{
				Query: "SELECT+key_id,outcome+FROM+key_verifications_v1+WHERE+outcome='VALID'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithData[openapi.V2AnalyticsGetVerificationsRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
