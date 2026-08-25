package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestWhoami(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysWhoamiRequestBody
	}{
		{
			name: "with key flag",
			args: "keys whoami --key=sk_1234abcdef5678",
			want: openapi.V2KeysWhoamiRequestBody{
				Key: "sk_1234abcdef5678",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequest[openapi.V2KeysWhoamiRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
