package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetPermission(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsGetPermissionRequestBody
	}{
		{
			name: "by permission id",
			args: "permissions get-permission --permission=perm_1234567890abcdef",
			want: openapi.V2PermissionsGetPermissionRequestBody{
				Permission: "perm_1234567890abcdef",
			},
		},
		{
			name: "different permission id",
			args: "permissions get-permission --permission=perm_abc123",
			want: openapi.V2PermissionsGetPermissionRequestBody{
				Permission: "perm_abc123",
			},
		},
		{
			name: "permission name with dots and hyphens",
			args: "permissions get-permission --permission=perm_my-org.read-v2",
			want: openapi.V2PermissionsGetPermissionRequestBody{
				Permission: "perm_my-org.read-v2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2PermissionsGetPermissionRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
