package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysGetKeyRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "keys get-key --key-id=key_1234abcd",
			want: openapi.V2KeysGetKeyRequestBody{
				KeyId:   "key_1234abcd",
				Decrypt: ptr.P(false),
			},
		},
		{
			name: "with decrypt flag",
			args: "keys get-key --key-id=key_1234abcd --decrypt",
			want: openapi.V2KeysGetKeyRequestBody{
				KeyId:   "key_1234abcd",
				Decrypt: ptr.P(true),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysGetKeyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
