package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateRole(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsCreateRoleRequestBody
	}{
		{
			name: "minimal required flags",
			args: "permissions create-role --name=content.editor",
			want: openapi.V2PermissionsCreateRoleRequestBody{
				Name: "content.editor",
			},
		},
		{
			name: "with description",
			args: "permissions create-role --name=billing.manager --description=manages-billing-resources",
			want: openapi.V2PermissionsCreateRoleRequestBody{
				Name:        "billing.manager",
				Description: ptr.P("manages-billing-resources"),
			},
		},
		{
			name: "simple role name",
			args: "permissions create-role --name=admin",
			want: openapi.V2PermissionsCreateRoleRequestBody{
				Name: "admin",
			},
		},
		{
			name: "role with underscores",
			args: "permissions create-role --name=api_reader",
			want: openapi.V2PermissionsCreateRoleRequestBody{
				Name: "api_reader",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2PermissionsCreateRoleRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
