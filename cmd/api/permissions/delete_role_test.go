package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeleteRole(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsDeleteRoleRequestBody
	}{
		{
			name: "delete by role ID",
			args: "permissions delete-role --role=role_dns_manager",
			want: openapi.V2PermissionsDeleteRoleRequestBody{
				Role: "role_dns_manager",
			},
		},
		{
			name: "delete by role name",
			args: "permissions delete-role --role=admin",
			want: openapi.V2PermissionsDeleteRoleRequestBody{
				Role: "admin",
			},
		},
		{
			name: "delete by dotted role name",
			args: "permissions delete-role --role=support.readonly",
			want: openapi.V2PermissionsDeleteRoleRequestBody{
				Role: "support.readonly",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2PermissionsDeleteRoleRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
