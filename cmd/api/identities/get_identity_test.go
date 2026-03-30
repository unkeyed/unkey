package identities

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetIdentity(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2IdentitiesGetIdentityRequestBody
	}{
		{
			name: "by external id",
			args: "identities get-identity --identity=user_123",
			want: openapi.V2IdentitiesGetIdentityRequestBody{
				Identity: "user_123",
			},
		},
		{
			name: "by identity id",
			args: "identities get-identity --identity=id_abc123",
			want: openapi.V2IdentitiesGetIdentityRequestBody{
				Identity: "id_abc123",
			},
		},
		{
			name: "with dots and hyphens",
			args: "identities get-identity --identity=org.acme-user-42",
			want: openapi.V2IdentitiesGetIdentityRequestBody{
				Identity: "org.acme-user-42",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2IdentitiesGetIdentityRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
