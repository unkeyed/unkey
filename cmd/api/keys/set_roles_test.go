package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestSetRoles(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysSetRolesRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "keys set-roles --key-id=key_1234abcd --roles=admin",
			want: openapi.V2KeysSetRolesRequestBody{
				KeyId: "key_1234abcd",
				Roles: []string{"admin"},
			},
		},
		{
			name: "multiple roles",
			args: "keys set-roles --key-id=key_1234abcd --roles=admin,billing_reader,api_admin",
			want: openapi.V2KeysSetRolesRequestBody{
				KeyId: "key_1234abcd",
				Roles: []string{"admin", "billing_reader", "api_admin"},
			},
		},
	}

	arrayResponse := `{"meta":{"requestId":"test"},"data":[]}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2KeysSetRolesRequestBody](t, Cmd(), tt.args, arrayResponse)
			require.Equal(t, tt.want, req)
		})
	}
}
