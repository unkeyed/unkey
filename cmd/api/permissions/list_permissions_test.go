package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListPermissions(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsListPermissionsRequestBody
	}{
		{
			name: "minimal",
			args: "permissions list-permissions",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit: new(100),
			},
		},
		{
			name: "with limit",
			args: "permissions list-permissions --limit=50",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit: new(50),
			},
		},
		{
			name: "with cursor",
			args: "permissions list-permissions --cursor=eyJrZXkiOiJwZXJtXzEyMzQifQ==",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit:  new(100),
				Cursor: new("eyJrZXkiOiJwZXJtXzEyMzQifQ=="),
			},
		},
		{
			name: "with limit and cursor",
			args: "permissions list-permissions --limit=25 --cursor=eyJrZXkiOiJwZXJtXzU2NzgifQ==",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit:  new(25),
				Cursor: new("eyJrZXkiOiJwZXJtXzU2NzgifQ=="),
			},
		},
		{
			name: "with search",
			args: "permissions list-permissions --search=documents",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit:  new(100),
				Search: new("documents"),
			},
		},
		{
			name: "with all flags",
			args: "permissions list-permissions --limit=25 --cursor=cursor_123 --search=documents",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit:  new(25),
				Cursor: new("cursor_123"),
				Search: new("documents"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[openapi.V2PermissionsListPermissionsRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
