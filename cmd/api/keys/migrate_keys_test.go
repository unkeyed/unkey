package keys

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestMigrateKeys(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2KeysMigrateKeysRequestBody
	}{
		{
			name: "with migration-id, api-id, and keys-json",
			args: `keys migrate-keys --migration-id=acme_migration --api-id=api_123456789 --keys-json=[{"hash":"abc123","enabled":true}]`,
			want: openapi.V2KeysMigrateKeysRequestBody{
				MigrationId: "acme_migration",
				ApiId:       "api_123456789",
				Keys: []openapi.V2KeysMigrateKeyData{
					{
						Hash:    "abc123",
						Enabled: ptr.P(true),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2KeysMigrateKeysRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
