package identities

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateIdentity(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2IdentitiesUpdateIdentityRequestBody
	}{
		{
			name: "minimal",
			args: "identities update-identity --identity=user_123",
			want: openapi.V2IdentitiesUpdateIdentityRequestBody{
				Identity: "user_123",
			},
		},
		{
			name: "with meta-json",
			args: `identities update-identity --identity=user_123 --meta-json={"plan":"premium","name":"Alice"}`,
			want: openapi.V2IdentitiesUpdateIdentityRequestBody{
				Identity: "user_123",
				Meta:     ptr.P(map[string]any{"plan": "premium", "name": "Alice"}),
			},
		},
		{
			name: "with ratelimits-json",
			args: `identities update-identity --identity=user_123 --ratelimits-json=[{"name":"requests","limit":1000,"duration":3600000,"autoApply":true}]`,
			want: openapi.V2IdentitiesUpdateIdentityRequestBody{
				Identity: "user_123",
				Ratelimits: ptr.P([]openapi.RatelimitRequest{
					{
						Name:      "requests",
						Limit:     1000,
						Duration:  3600000,
						AutoApply: true,
					},
				}),
			},
		},
		{
			name: "all flags",
			args: `identities update-identity --identity=user_123 --meta-json={"tier":"enterprise"} --ratelimits-json=[{"name":"api","limit":500,"duration":60000,"autoApply":false}]`,
			want: openapi.V2IdentitiesUpdateIdentityRequestBody{
				Identity: "user_123",
				Meta:     ptr.P(map[string]any{"tier": "enterprise"}),
				Ratelimits: ptr.P([]openapi.RatelimitRequest{
					{
						Name:      "api",
						Limit:     500,
						Duration:  60000,
						AutoApply: false,
					},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2IdentitiesUpdateIdentityRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
