package permissions

import (
	"encoding/json"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdatePermission(t *testing.T) {
	permissionID := uid.New(uid.PermissionPrefix)
	args := "permissions update-permission --permission=" + permissionID
	tests := []struct {
		name string
		args string
		body bool
		want openapi.V2PermissionsUpdatePermissionRequestBody
	}{
		{
			name: "omitted fields remain unset",
			args: args,
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission: permissionID,
			},
		},
		{
			name: "rename by ID",
			args: args + ` --name="Read documents"`,
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission: permissionID,
				Name:       new("Read documents"),
			},
		},
		{
			name: "update slug by current slug",
			args: "permissions update-permission --permission=documents-read --slug=documents.read",
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission: "documents-read",
				Slug:       new("documents.read"),
			},
		},
		{
			name: "update description",
			args: args + ` --description="Read document resources"`,
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission:  permissionID,
				Description: nullable.NewNullableWithValue("Read document resources"),
			},
		},
		{
			name: "empty description clears it",
			args: args + " --description=",
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission:  permissionID,
				Description: nullable.NewNullableWithValue(""),
			},
		},
		{
			name: "empty name is preserved for API validation",
			args: args + " --name=",
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission: permissionID,
				Name:       new(""),
			},
		},
		{
			name: "empty slug is preserved for API validation",
			args: args + " --slug=",
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission: permissionID,
				Slug:       new(""),
			},
		},
		{
			name: "all flags",
			args: args + ` --name="Read documents" --slug=documents.read --description="Read document resources"`,
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission:  permissionID,
				Name:        new("Read documents"),
				Slug:        new("documents.read"),
				Description: nullable.NewNullableWithValue("Read document resources"),
			},
		},
		{
			name: "body preserves omitted description",
			body: true,
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission: permissionID,
				Name:       new("Read documents"),
			},
		},
		{
			name: "body preserves explicit null description",
			body: true,
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission:  permissionID,
				Description: nullable.NewNullNullable[string](),
			},
		},
		{
			name: "body preserves empty description",
			body: true,
			want: openapi.V2PermissionsUpdatePermissionRequestBody{
				Permission:  permissionID,
				Description: nullable.NewNullableWithValue(""),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			if tt.body {
				body, err := json.Marshal(tt.want)
				require.NoError(t, err)
				args = "permissions update-permission --body='" + string(body) + "'"
			}
			got := testutil.CaptureRequest[openapi.V2PermissionsUpdatePermissionRequestBody](t, Cmd(), args)
			require.Equal(t, tt.want, got)
		})
	}
}
