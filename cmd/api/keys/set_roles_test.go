package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestSetRoles(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysSetRolesRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "keys set-roles --key-id=key_1234abcd --roles=admin",
			want: openapi.V2KeysSetRolesRequestBody{
				KeyId: "key_1234abcd",
				Roles: []string{"admin"},
			},
		},
		{
			name: "multiple roles",
			args: "keys set-roles --key-id=key_1234abcd --roles=admin,billing_reader,api_admin",
			want: openapi.V2KeysSetRolesRequestBody{
				KeyId: "key_1234abcd",
				Roles: []string{"admin", "billing_reader", "api_admin"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[openapi.V2KeysSetRolesRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
