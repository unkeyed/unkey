package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestRerollKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysRerollKeyRequestBody
	}{
		{
			name: "with key-id and immediate expiration",
			args: "keys reroll-key --key-id=key_1234abcd --expiration=0",
			want: openapi.V2KeysRerollKeyRequestBody{
				KeyId:      "key_1234abcd",
				Expiration: 0,
			},
		},
		{
			name: "with key-id and 24h expiration",
			args: "keys reroll-key --key-id=key_1234abcd --expiration=86400000",
			want: openapi.V2KeysRerollKeyRequestBody{
				KeyId:      "key_1234abcd",
				Expiration: 86400000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysRerollKeyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
