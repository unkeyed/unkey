package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestRemovePermissions(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysRemovePermissionsRequestBody
	}{
		{
			name: "minimal with key-id and permissions",
			args: "keys remove-permissions --key-id=key_1234abcd --permissions=documents.read,documents.write",
			want: openapi.V2KeysRemovePermissionsRequestBody{
				KeyId:       "key_1234abcd",
				Permissions: []string{"documents.read", "documents.write"},
			},
		},
	}

	arrayResponse := `{"meta":{"requestId":"test"},"data":[]}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2KeysRemovePermissionsRequestBody](t, Cmd(), tt.args, arrayResponse)
			require.Equal(t, tt.want, req)
		})
	}
}
