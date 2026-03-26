package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestSetPermissions(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysSetPermissionsRequestBody
	}{
		{
			name: "minimal with key-id and permissions",
			args: "keys set-permissions --key-id=key_1234abcd --permissions=documents.read,documents.write,settings.view",
			want: openapi.V2KeysSetPermissionsRequestBody{
				KeyId:       "key_1234abcd",
				Permissions: []string{"documents.read", "documents.write", "settings.view"},
			},
		},
	}

	arrayResponse := `{"meta":{"requestId":"test"},"data":[]}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2KeysSetPermissionsRequestBody](t, Cmd(), tt.args, arrayResponse)
			require.Equal(t, tt.want, req)
		})
	}
}
