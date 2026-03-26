package keys

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateCredits(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysUpdateCreditsRequestBody
	}{
		{
			name: "minimal set operation without value (unlimited)",
			args: "keys update-credits --key-id=key_1234abcd --operation=set",
			want: openapi.V2KeysUpdateCreditsRequestBody{
				KeyId:     "key_1234abcd",
				Operation: openapi.Set,
			},
		},
		{
			name: "set operation with value",
			args: "keys update-credits --key-id=key_1234abcd --operation=set --value=1000",
			want: openapi.V2KeysUpdateCreditsRequestBody{
				KeyId:     "key_1234abcd",
				Operation: openapi.Set,
				Value:     nullable.NewNullableWithValue(int64(1000)),
			},
		},
		{
			name: "increment operation with value",
			args: "keys update-credits --key-id=key_1234abcd --operation=increment --value=500",
			want: openapi.V2KeysUpdateCreditsRequestBody{
				KeyId:     "key_1234abcd",
				Operation: openapi.Increment,
				Value:     nullable.NewNullableWithValue(int64(500)),
			},
		},
		{
			name: "decrement operation with value",
			args: "keys update-credits --key-id=key_1234abcd --operation=decrement --value=100",
			want: openapi.V2KeysUpdateCreditsRequestBody{
				KeyId:     "key_1234abcd",
				Operation: openapi.Decrement,
				Value:     nullable.NewNullableWithValue(int64(100)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysUpdateCreditsRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
