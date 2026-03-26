package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestMultiLimit(t *testing.T) {
	tests := []struct {
		name string
		args string
		want []openapi.V2RatelimitLimitRequestBody
	}{
		{
			name: "single limit",
			args: `ratelimit multi-limit --limits-json=[{"namespace":"api.requests","identifier":"user_abc123","limit":100,"duration":60000}]`,
			want: []openapi.V2RatelimitLimitRequestBody{
				{
					Namespace:  "api.requests",
					Identifier: "user_abc123",
					Limit:      100,
					Duration:   60000,
					Cost:       ptr.P(int64(1)),
				},
			},
		},
		{
			name: "two limits with cost",
			args: `ratelimit multi-limit --limits-json=[{"namespace":"api.light_operations","identifier":"user_xyz789","limit":100,"duration":60000,"cost":1},{"namespace":"api.heavy_operations","identifier":"user_xyz789","limit":50,"duration":3600000,"cost":5}]`,
			want: []openapi.V2RatelimitLimitRequestBody{
				{
					Namespace:  "api.light_operations",
					Identifier: "user_xyz789",
					Limit:      100,
					Duration:   60000,
					Cost:       ptr.P(int64(1)),
				},
				{
					Namespace:  "api.heavy_operations",
					Identifier: "user_xyz789",
					Limit:      50,
					Duration:   3600000,
					Cost:       ptr.P(int64(5)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[[]openapi.V2RatelimitLimitRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
