package keys

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
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
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "explicit enabled false",
			args: "keys create-key --api-id=api_123 --enabled=false",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ByteLength:  new(16),
				Enabled:     new(false),
				Recoverable: new(false),
			},
		},
		{
			name: "recoverable key",
			args: "keys create-key --api-id=api_123 --recoverable",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(true),
			},
		},
		{
			name: "with prefix and name",
			args: "keys create-key --api-id=api_123 --prefix=sk --name=production",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Prefix:      new("sk"),
				Name:        new("production"),
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "custom byte length",
			args: "keys create-key --api-id=api_123 --byte-length=32",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ByteLength:  new(32),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "with external id",
			args: "keys create-key --api-id=api_123 --external-id=user_456",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				ExternalId:  new("user_456"),
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "with expiration",
			args: "keys create-key --api-id=api_123 --expires=1700000000000",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Expires:     new(int64(1700000000000)),
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "with roles and permissions",
			args: "keys create-key --api-id=api_123 --roles=admin,reader --permissions=docs.read,docs.write",
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Roles:       new([]string{"admin", "reader"}),
				Permissions: new([]string{"docs.read", "docs.write"}),
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "with metadata json",
			args: `keys create-key --api-id=api_123 --meta='{"plan":"pro","org":"acme"}'`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Meta:        new(map[string]any{"plan": "pro", "org": "acme"}),
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "with credits json",
			args: `keys create-key --api-id=api_123 --credits='{"remaining":1000}'`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId: "api_123",
				Credits: &openapi.KeyCreditsData{
					Remaining: nullable.NewNullableWithValue(int64(1000)),
				},
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "with ratelimits json",
			args: `keys create-key --api-id=api_123 --ratelimits='[{"name":"req","limit":100,"duration":60000,"autoApply":true}]'`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId: "api_123",
				Ratelimits: new([]openapi.RatelimitRequest{
					{Name: "req", Limit: 100, Duration: 60000, AutoApply: true},
				}),
				ByteLength:  new(16),
				Enabled:     new(true),
				Recoverable: new(false),
			},
		},
		{
			name: "all flags",
			args: `keys create-key --api-id=api_123 --prefix=sk --name=test --byte-length=32 --external-id=user_456 --expires=1700000000000 --enabled=false --recoverable --roles=admin,reader --permissions=docs.read,docs.write --meta='{"plan":"pro"}' --credits='{"remaining":1000}' --ratelimits='[{"name":"req","limit":100,"duration":60000,"autoApply":false}]'`,
			want: openapi.V2KeysCreateKeyRequestBody{
				ApiId:       "api_123",
				Prefix:      new("sk"),
				Name:        new("test"),
				ByteLength:  new(32),
				ExternalId:  new("user_456"),
				Expires:     new(int64(1700000000000)),
				Enabled:     new(false),
				Recoverable: new(true),
				Roles:       new([]string{"admin", "reader"}),
				Permissions: new([]string{"docs.read", "docs.write"}),
				Meta:        new(map[string]any{"plan": "pro"}),
				Credits: &openapi.KeyCreditsData{
					Remaining: nullable.NewNullableWithValue(int64(1000)),
				},
				Ratelimits: new([]openapi.RatelimitRequest{
					{Name: "req", Limit: 100, Duration: 60000, AutoApply: false},
				}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequest[openapi.V2KeysCreateKeyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
