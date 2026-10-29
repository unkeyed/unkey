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

func TestUpdateRole(t *testing.T) {
	roleID := uid.New(uid.RolePrefix)
	args := "permissions update-role --role=" + roleID
	tests := []struct {
		name string
		args string
		body bool
		want openapi.V2PermissionsUpdateRoleRequestBody
	}{
		{
			name: "omitted fields remain unset",
			args: args,
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role: roleID,
			},
		},
		{
			name: "rename by ID",
			args: args + " --name=support.readonly",
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role: roleID,
				Name: new("support.readonly"),
			},
		},
		{
			name: "update description by current name",
			args: `permissions update-role --role=support.readonly --description="Read-only support access"`,
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role:        "support.readonly",
				Description: nullable.NewNullableWithValue("Read-only support access"),
			},
		},
		{
			name: "empty description clears it",
			args: args + " --description=",
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role:        roleID,
				Description: nullable.NewNullableWithValue(""),
			},
		},
		{
			name: "empty name is preserved for API validation",
			args: args + " --name=",
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role: roleID,
				Name: new(""),
			},
		},
		{
			name: "all flags",
			args: args + ` --name=support.readonly --description="Read-only support access"`,
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role:        roleID,
				Name:        new("support.readonly"),
				Description: nullable.NewNullableWithValue("Read-only support access"),
			},
		},
		{
			name: "body preserves omitted description",
			body: true,
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role: roleID,
				Name: new("support.readonly"),
			},
		},
		{
			name: "body preserves explicit null description",
			body: true,
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role:        roleID,
				Description: nullable.NewNullNullable[string](),
			},
		},
		{
			name: "body preserves empty description",
			body: true,
			want: openapi.V2PermissionsUpdateRoleRequestBody{
				Role:        roleID,
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
				args = "permissions update-role --body='" + string(body) + "'"
			}
			got := testutil.CaptureRequest[openapi.V2PermissionsUpdateRoleRequestBody](t, Cmd(), args)
			require.Equal(t, tt.want, got)
		})
	}
}
