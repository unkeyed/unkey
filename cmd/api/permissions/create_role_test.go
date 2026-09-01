package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestCreateRole(t *testing.T) {
	tests := []struct {
		name string
		args string
		want components.V2PermissionsCreateRoleRequestBody
	}{
		{
			name: "minimal required flags",
			args: "permissions create-role --name=content.editor",
			want: components.V2PermissionsCreateRoleRequestBody{
				Name: "content.editor",
			},
		},
		{
			name: "with description",
			args: "permissions create-role --name=billing.manager --description=manages-billing-resources",
			want: components.V2PermissionsCreateRoleRequestBody{
				Name:        "billing.manager",
				Description: ptr.P("manages-billing-resources"),
			},
		},
		{
			name: "with permissions",
			args: "permissions create-role --name=content.editor --permissions=documents.read,documents.write",
			want: components.V2PermissionsCreateRoleRequestBody{
				Name:        "content.editor",
				Permissions: []string{"documents.read", "documents.write"},
			},
		},
		{
			name: "simple role name",
			args: "permissions create-role --name=admin",
			want: components.V2PermissionsCreateRoleRequestBody{
				Name: "admin",
			},
		},
		{
			name: "role with underscores",
			args: "permissions create-role --name=api_reader",
			want: components.V2PermissionsCreateRoleRequestBody{
				Name: "api_reader",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequest[components.V2PermissionsCreateRoleRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
