package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeletePermission(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsDeletePermissionRequestBody
	}{
		{
			name: "delete by permission ID",
			args: "permissions delete-permission --permission=perm_1234567890abcdef",
			want: openapi.V2PermissionsDeletePermissionRequestBody{
				Permission: "perm_1234567890abcdef",
			},
		},
		{
			name: "delete by slug",
			args: "permissions delete-permission --permission=documents.read",
			want: openapi.V2PermissionsDeletePermissionRequestBody{
				Permission: "documents.read",
			},
		},
		{
			name: "delete by hierarchical slug",
			args: "permissions delete-permission --permission=admin.users.delete",
			want: openapi.V2PermissionsDeletePermissionRequestBody{
				Permission: "admin.users.delete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2PermissionsDeletePermissionRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
