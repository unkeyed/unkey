package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetOverride(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitGetOverrideRequestBody
	}{
		{
			name: "basic",
			args: "ratelimit get-override --namespace=api.requests --identifier=premium_user_123",
			want: openapi.V2RatelimitGetOverrideRequestBody{
				Namespace:  "api.requests",
				Identifier: "premium_user_123",
			},
		},
		{
			name: "wildcard identifier",
			args: "ratelimit get-override --namespace=api.requests --identifier=premium_*",
			want: openapi.V2RatelimitGetOverrideRequestBody{
				Namespace:  "api.requests",
				Identifier: "premium_*",
			},
		},
		{
			name: "dotted namespace",
			args: "ratelimit get-override --namespace=billing.v2 --identifier=org_acme",
			want: openapi.V2RatelimitGetOverrideRequestBody{
				Namespace:  "billing.v2",
				Identifier: "org_acme",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2RatelimitGetOverrideRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
