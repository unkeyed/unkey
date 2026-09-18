package permissions

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestSetRolePermissions(t *testing.T) {
	tests := []struct {
		name string
		args string
		want map[string]any
	}{
		{
			name: "set permissions",
			args: "permissions set-role-permissions --role=admin --permissions=documents.read,documents.write",
			want: map[string]any{"role": "admin", "permissions": []any{"documents.read", "documents.write"}},
		},
		{
			name: "clear permissions",
			args: "permissions set-role-permissions --role-id=admin --permissions=",
			want: map[string]any{"roleId": "admin", "permissions": []any{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[map[string]any](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
