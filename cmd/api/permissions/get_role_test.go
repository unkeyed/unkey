package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetRole(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsGetRoleRequestBody
	}{
		{
			name: "by role id",
			args: "permissions get-role --role=role_1234567890abcdef",
			want: openapi.V2PermissionsGetRoleRequestBody{
				Role: "role_1234567890abcdef",
			},
		},
		{
			name: "by role name",
			args: "permissions get-role --role=admin",
			want: openapi.V2PermissionsGetRoleRequestBody{
				Role: "admin",
			},
		},
		{
			name: "role name with hyphens",
			args: "permissions get-role --role=billing-manager",
			want: openapi.V2PermissionsGetRoleRequestBody{
				Role: "billing-manager",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2PermissionsGetRoleRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
