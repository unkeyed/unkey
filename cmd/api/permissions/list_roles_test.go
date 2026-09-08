package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListRoles(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsListRolesRequestBody
	}{
		{
			name: "minimal",
			args: "permissions list-roles",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit: new(100),
			},
		},
		{
			name: "with limit",
			args: "permissions list-roles --limit=50",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit: new(50),
			},
		},
		{
			name: "with cursor",
			args: "permissions list-roles --cursor=eyJrZXkiOiJyb2xlXzEyMzQifQ==",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit:  new(100),
				Cursor: new("eyJrZXkiOiJyb2xlXzEyMzQifQ=="),
			},
		},
		{
			name: "with limit and cursor",
			args: "permissions list-roles --limit=25 --cursor=eyJrZXkiOiJyb2xlXzU2NzgifQ==",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit:  new(25),
				Cursor: new("eyJrZXkiOiJyb2xlXzU2NzgifQ=="),
			},
		},
		{
			name: "with search",
			args: "permissions list-roles --search=admin",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit:  new(100),
				Search: new("admin"),
			},
		},
		{
			name: "with all flags",
			args: "permissions list-roles --limit=25 --cursor=cursor_123 --search=admin",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit:  new(25),
				Cursor: new("cursor_123"),
				Search: new("admin"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[openapi.V2PermissionsListRolesRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
