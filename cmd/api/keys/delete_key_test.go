package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeleteKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysDeleteKeyRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "keys delete-key --key-id=key_1234abcd",
			want: openapi.V2KeysDeleteKeyRequestBody{
				KeyId:     "key_1234abcd",
				Permanent: ptr.P(false),
			},
		},
		{
			name: "with permanent flag",
			args: "keys delete-key --key-id=key_1234abcd --permanent",
			want: openapi.V2KeysDeleteKeyRequestBody{
				KeyId:     "key_1234abcd",
				Permanent: ptr.P(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysDeleteKeyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
