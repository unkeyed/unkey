package identities

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeleteIdentity(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2IdentitiesDeleteIdentityRequestBody
	}{
		{
			name: "delete by identity id",
			args: "identities delete-identity --identity=id_abc123",
			want: openapi.V2IdentitiesDeleteIdentityRequestBody{
				Identity: "id_abc123",
			},
		},
		{
			name: "delete by external id",
			args: "identities delete-identity --identity=user_123",
			want: openapi.V2IdentitiesDeleteIdentityRequestBody{
				Identity: "user_123",
			},
		},
		{
			name: "identity with dots and hyphens",
			args: "identities delete-identity --identity=org.team-member-42",
			want: openapi.V2IdentitiesDeleteIdentityRequestBody{
				Identity: "org.team-member-42",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2IdentitiesDeleteIdentityRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
