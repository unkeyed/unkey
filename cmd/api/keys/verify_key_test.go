package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestVerifyKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysVerifyKeyRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "keys verify-key --key=sk_1234abcdef",
			want: openapi.V2KeysVerifyKeyRequestBody{
				Key: "sk_1234abcdef",
			},
		},
		{
			name: "with permissions",
			args: "keys verify-key --key=sk_1234abcdef --permissions=documents.read",
			want: openapi.V2KeysVerifyKeyRequestBody{
				Key:         "sk_1234abcdef",
				Permissions: ptr.P("documents.read"),
			},
		},
		{
			name: "with tags",
			args: "keys verify-key --key=sk_1234abcdef --tags=endpoint=/users/profile,method=GET",
			want: openapi.V2KeysVerifyKeyRequestBody{
				Key:  "sk_1234abcdef",
				Tags: ptr.P([]string{"endpoint=/users/profile", "method=GET"}),
			},
		},
		{
			name: "with credits json",
			args: `keys verify-key --key=sk_1234abcdef --credits-json={"cost":5}`,
			want: openapi.V2KeysVerifyKeyRequestBody{
				Key: "sk_1234abcdef",
				Credits: &openapi.KeysVerifyKeyCredits{
					Cost: 5,
				},
			},
		},
		{
			name: "with ratelimits json",
			args: `keys verify-key --key=sk_1234abcdef --ratelimits-json=[{"name":"requests","limit":100,"duration":60000}]`,
			want: openapi.V2KeysVerifyKeyRequestBody{
				Key: "sk_1234abcdef",
				Ratelimits: ptr.P([]openapi.KeysVerifyKeyRatelimit{
					{Name: "requests", Limit: ptr.P(100), Duration: ptr.P(60000), Cost: ptr.P(1)},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysVerifyKeyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
