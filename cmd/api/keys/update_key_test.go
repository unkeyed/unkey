package keys

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateKey(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysUpdateKeyRequestBody
	}{
		{
			name: "only required field",
			args: "keys update-key --key-id=key_123",
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId: "key_123",
			},
		},
		{
			name: "enabled false",
			args: "keys update-key --key-id=key_123 --enabled=false",
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId:   "key_123",
				Enabled: ptr.P(false),
			},
		},
		{
			name: "enabled true",
			args: "keys update-key --key-id=key_123 --enabled=true",
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId:   "key_123",
				Enabled: ptr.P(true),
			},
		},
		{
			name: "enabled omitted does not send enabled",
			args: "keys update-key --key-id=key_123 --name=updated",
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId: "key_123",
				Name:  nullable.NewNullableWithValue("updated"),
			},
		},
		{
			name: "with external id",
			args: "keys update-key --key-id=key_123 --external-id=user_789",
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId:      "key_123",
				ExternalId: nullable.NewNullableWithValue("user_789"),
			},
		},
		{
			name: "with roles and permissions",
			args: "keys update-key --key-id=key_123 --roles=admin,billing --permissions=docs.read",
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId:       "key_123",
				Roles:       ptr.P([]string{"admin", "billing"}),
				Permissions: ptr.P([]string{"docs.read"}),
			},
		},
		{
			name: "with metadata json",
			args: `keys update-key --key-id=key_123 --meta-json={"tier":"enterprise"}`,
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId: "key_123",
				Meta:  nullable.NewNullableWithValue(map[string]any{"tier": "enterprise"}),
			},
		},
		{
			name: "multiple fields at once",
			args: `keys update-key --key-id=key_123 --name=updated --enabled=false --roles=admin --meta-json={"plan":"pro"}`,
			want: openapi.V2KeysUpdateKeyRequestBody{
				KeyId:   "key_123",
				Name:    nullable.NewNullableWithValue("updated"),
				Enabled: ptr.P(false),
				Roles:   ptr.P([]string{"admin"}),
				Meta:    nullable.NewNullableWithValue(map[string]any{"plan": "pro"}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysUpdateKeyRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
