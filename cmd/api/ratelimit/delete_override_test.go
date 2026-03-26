package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeleteOverride(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitDeleteOverrideRequestBody
	}{
		{
			name: "basic",
			args: "ratelimit delete-override --namespace=api.requests --identifier=premium_user_123",
			want: openapi.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  "api.requests",
				Identifier: "premium_user_123",
			},
		},
		{
			name: "wildcard identifier",
			args: "ratelimit delete-override --namespace=api.requests --identifier=premium_*",
			want: openapi.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  "api.requests",
				Identifier: "premium_*",
			},
		},
		{
			name: "dotted namespace",
			args: "ratelimit delete-override --namespace=billing.v2 --identifier=org_acme",
			want: openapi.V2RatelimitDeleteOverrideRequestBody{
				Namespace:  "billing.v2",
				Identifier: "org_acme",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2RatelimitDeleteOverrideRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
