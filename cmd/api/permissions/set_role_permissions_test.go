package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestSetRolePermissions(t *testing.T) {
	tests := []struct {
		name string
		args string
		want components.V2PermissionsSetRolePermissionsRequestBody
	}{
		{
			name: "set permissions",
			args: "permissions set-role-permissions --role-id=role_1234abcd --permissions=documents.read,documents.write",
			want: components.V2PermissionsSetRolePermissionsRequestBody{
				RoleID:      "role_1234abcd",
				Permissions: []string{"documents.read", "documents.write"},
			},
		},
		{
			name: "clear permissions",
			args: "permissions set-role-permissions --role-id=admin --permissions=",
			want: components.V2PermissionsSetRolePermissionsRequestBody{
				RoleID:      "admin",
				Permissions: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[components.V2PermissionsSetRolePermissionsRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
