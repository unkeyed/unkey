package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestSetOverride(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitSetOverrideRequestBody
	}{
		{
			name: "basic",
			args: "ratelimit set-override --namespace=api.requests --identifier=premium_user_123 --limit=1000 --duration=60000",
			want: openapi.V2RatelimitSetOverrideRequestBody{
				Namespace:  "api.requests",
				Identifier: "premium_user_123",
				Limit:      1000,
				Duration:   60000,
			},
		},
		{
			name: "wildcard identifier",
			args: "ratelimit set-override --namespace=api.requests --identifier=premium_* --limit=500 --duration=60000",
			want: openapi.V2RatelimitSetOverrideRequestBody{
				Namespace:  "api.requests",
				Identifier: "premium_*",
				Limit:      500,
				Duration:   60000,
			},
		},
		{
			name: "low limit strict throttle",
			args: "ratelimit set-override --namespace=api.requests --identifier=throttled_user --limit=5 --duration=3600000",
			want: openapi.V2RatelimitSetOverrideRequestBody{
				Namespace:  "api.requests",
				Identifier: "throttled_user",
				Limit:      5,
				Duration:   3600000,
			},
		},
		{
			name: "high limit long duration",
			args: "ratelimit set-override --namespace=billing --identifier=enterprise_org_42 --limit=100000 --duration=86400000",
			want: openapi.V2RatelimitSetOverrideRequestBody{
				Namespace:  "billing",
				Identifier: "enterprise_org_42",
				Limit:      100000,
				Duration:   86400000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2RatelimitSetOverrideRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
