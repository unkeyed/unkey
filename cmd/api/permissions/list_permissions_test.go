package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListPermissions(t *testing.T) {
	listResponse := `{"meta":{"requestId":"test"},"data":[]}`

	tests := []struct {
		name string
		args string
		want openapi.V2PermissionsListPermissionsRequestBody
	}{
		{
			name: "minimal",
			args: "permissions list-permissions",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit: ptr.P(100),
			},
		},
		{
			name: "with limit",
			args: "permissions list-permissions --limit=50",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit: ptr.P(50),
			},
		},
		{
			name: "with cursor",
			args: "permissions list-permissions --cursor=eyJrZXkiOiJwZXJtXzEyMzQifQ==",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit:  ptr.P(100),
				Cursor: ptr.P("eyJrZXkiOiJwZXJtXzEyMzQifQ=="),
			},
		},
		{
			name: "with limit and cursor",
			args: "permissions list-permissions --limit=25 --cursor=eyJrZXkiOiJwZXJtXzU2NzgifQ==",
			want: openapi.V2PermissionsListPermissionsRequestBody{
				Limit:  ptr.P(25),
				Cursor: ptr.P("eyJrZXkiOiJwZXJtXzU2NzgifQ=="),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2PermissionsListPermissionsRequestBody](t, Cmd(), tt.args, listResponse)
			require.Equal(t, tt.want, req)
		})
	}
}
