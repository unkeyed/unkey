package keys

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysCreateKeyRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "keys create-key --api-id=api_123",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "explicit enabled false",
			args: "keys create-key --api-id=api_123 --enabled=false",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(false),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "recoverable key",
			args: "keys create-key --api-id=api_123 --recoverable",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(true),
			},
		},
		{
			name: "with prefix and name",
			args: "keys create-key --api-id=api_123 --prefix=sk --name=production",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Prefix:      ptr.P("sk"),
				Name:        ptr.P("production"),
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "custom byte length",
			args: "keys create-key --api-id=api_123 --byte-length=32",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ByteLength:  ptr.P(32),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "with external id",
			args: "keys create-key --api-id=api_123 --external-id=user_456",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ExternalId:  ptr.P("user_456"),
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "with expiration",
			args: "keys create-key --api-id=api_123 --expires=1700000000000",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Expires:     ptr.P(int64(1700000000000)),
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "with roles and permissions",
			args: "keys create-key --api-id=api_123 --roles=admin,reader --permissions=docs.read,docs.write",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Roles:       ptr.P([]string{"admin", "reader"}),
				Permissions: ptr.P([]string{"docs.read", "docs.write"}),
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "with metadata json",
			args: `keys create-key --api-id=api_123 --meta-json={"plan":"pro","org":"acme"}`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Meta:        ptr.P(map[string]any{"plan": "pro", "org": "acme"}),
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "with credits json",
			args: `keys create-key --api-id=api_123 --credits-json={"remaining":1000}`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId: "api_123",
				Credits: &openapi.KeyCreditsData{
					Remaining: nullable.NewNullableWithValue(int64(1000)),
				},
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "with ratelimits json",
			args: `keys create-key --api-id=api_123 --ratelimits-json=[{"name":"req","limit":100,"duration":60000,"autoApply":true}]`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId: "api_123",
				Ratelimits: ptr.P([]openapi.RatelimitRequest{
					{Name: "req", Limit: 100, Duration: 60000, AutoApply: true},
				}),
				ByteLength:  ptr.P(16),
				Enabled:     ptr.P(true),
				Recoverable: ptr.P(false),
			},
		},
		{
			name: "all flags",
			args: `keys create-key --api-id=api_123 --prefix=sk --name=test --byte-length=32 --external-id=user_456 --expires=1700000000000 --enabled=false --recoverable --roles=admin,reader --permissions=docs.read,docs.write --meta-json={"plan":"pro"} --credits-json={"remaining":1000} --ratelimits-json=[{"name":"req","limit":100,"duration":60000,"autoApply":false}]`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Prefix:      ptr.P("sk"),
				Name:        ptr.P("test"),
				ByteLength:  ptr.P(32),
				ExternalId:  ptr.P("user_456"),
				Expires:     ptr.P(int64(1700000000000)),
				Enabled:     ptr.P(false),
				Recoverable: ptr.P(true),
				Roles:       ptr.P([]string{"admin", "reader"}),
				Permissions: ptr.P([]string{"docs.read", "docs.write"}),
				Meta:        ptr.P(map[string]any{"plan": "pro"}),
				Credits: &openapi.KeyCreditsData{
					Remaining: nullable.NewNullableWithValue(int64(1000)),
				},
				Ratelimits: ptr.P([]openapi.RatelimitRequest{
					{Name: "req", Limit: 100, Duration: 60000, AutoApply: false},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysCreateKeyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
