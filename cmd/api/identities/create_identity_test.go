package identities

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateIdentity(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2IdentitiesCreateIdentityRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "identities create-identity --external-id=user_123",
			want: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "user_123",
			},
		},
		{
			name: "with meta json",
			args: `identities create-identity --external-id=user_123 --meta-json={"email":"alice@acme.com","plan":"premium"}`,
			want: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "user_123",
				Meta:       ptr.P(map[string]interface{}{"email": "alice@acme.com", "plan": "premium"}),
			},
		},
		{
			name: "with ratelimits json",
			args: `identities create-identity --external-id=user_123 --ratelimits-json=[{"name":"requests","limit":1000,"duration":60000,"autoApply":false}]`,
			want: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "user_123",
				Ratelimits: &[]openapi.RatelimitRequest{
					{Name: "requests", Limit: 1000, Duration: 60000, AutoApply: false},
				},
			},
		},
		{
			name: "all flags",
			args: `identities create-identity --external-id=user_123 --meta-json={"email":"alice@acme.com","plan":"premium"} --ratelimits-json=[{"name":"requests","limit":1000,"duration":60000,"autoApply":true}]`,
			want: openapi.V2IdentitiesCreateIdentityRequestBody{
				ExternalId: "user_123",
				Meta:       ptr.P(map[string]interface{}{"email": "alice@acme.com", "plan": "premium"}),
				Ratelimits: &[]openapi.RatelimitRequest{
					{Name: "requests", Limit: 1000, Duration: 60000, AutoApply: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2IdentitiesCreateIdentityRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
