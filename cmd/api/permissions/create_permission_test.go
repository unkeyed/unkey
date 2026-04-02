package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreatePermission(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsCreatePermissionRequestBody
	}{
		{
			name: "minimal required flags",
			args: "permissions create-permission --name=users.read --slug=users-read",
			want: openapi.V2PermissionsCreatePermissionRequestBody{
				Name: "users.read",
				Slug: "users-read",
			},
		},
		{
			name: "with description",
			args: "permissions create-permission --name=billing.write --slug=billing-write --description=write-access-to-billing",
			want: openapi.V2PermissionsCreatePermissionRequestBody{
				Name:        "billing.write",
				Slug:        "billing-write",
				Description: ptr.P("write-access-to-billing"),
			},
		},
		{
			name: "hierarchical name with dots",
			args: "permissions create-permission --name=admin.users.delete --slug=admin-users-delete",
			want: openapi.V2PermissionsCreatePermissionRequestBody{
				Name: "admin.users.delete",
				Slug: "admin-users-delete",
			},
		},
		{
			name: "slug with underscores",
			args: "permissions create-permission --name=analytics.view --slug=analytics_view",
			want: openapi.V2PermissionsCreatePermissionRequestBody{
				Name: "analytics.view",
				Slug: "analytics_view",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2PermissionsCreatePermissionRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
