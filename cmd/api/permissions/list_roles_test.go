package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListRoles(t *testing.T) {
	listResponse := `{"meta":{"requestId":"test"},"data":[]}`

	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsListRolesRequestBody
	}{
		{
			name: "minimal",
			args: "permissions list-roles",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit: ptr.P(100),
			},
		},
		{
			name: "with limit",
			args: "permissions list-roles --limit=50",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit: ptr.P(50),
			},
		},
		{
			name: "with cursor",
			args: "permissions list-roles --cursor=eyJrZXkiOiJyb2xlXzEyMzQifQ==",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit:  ptr.P(100),
				Cursor: ptr.P("eyJrZXkiOiJyb2xlXzEyMzQifQ=="),
			},
		},
		{
			name: "with limit and cursor",
			args: "permissions list-roles --limit=25 --cursor=eyJrZXkiOiJyb2xlXzU2NzgifQ==",
			want: openapi.V2PermissionsListRolesRequestBody{
				Limit:  ptr.P(25),
				Cursor: ptr.P("eyJrZXkiOiJyb2xlXzU2NzgifQ=="),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2PermissionsListRolesRequestBody](t, Cmd(), tt.args, listResponse)
			require.Equal(t, tt.want, req)
		})
	}
}
