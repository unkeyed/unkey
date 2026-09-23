package migration_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac/migration"
	"github.com/unkeyed/unkey/pkg/urn"
)

// TestProductionPermissions checks every exported slug against an independently
// specified outcome. Resource ownership is synthetic, not production metadata.
func TestProductionPermissions(t *testing.T) {
	require.Len(t, productionPermissions, 17762)
	translated, unsupported := 0, 0
	for _, tt := range productionPermissions {
		t.Run(tt.input, func(t *testing.T) {
			scope := migration.Scope{WorkspaceID: "ws_fixture"}
			parts := strings.Split(tt.input, ".")
			if len(parts) == 3 {
				scope.APIs = map[string]urn.V1{
					parts[1]: {WorkspaceID: "ws_fixture", Resource: "projects/proj_fixture/keyspaces/ks_fixture"},
				}
				scope.Namespaces = map[string]urn.V1{parts[1]: {WorkspaceID: "ws_fixture", Resource: "projects/proj_fixture/ratelimits/namespaces/" + parts[1]}}
				scope.Apps = map[string]urn.V1{parts[1]: {WorkspaceID: "ws_fixture", Resource: "projects/proj_fixture/apps/" + parts[1]}}
				scope.Environments = map[string]urn.V1{parts[1]: {WorkspaceID: "ws_fixture", Resource: "projects/proj_fixture/apps/app_fixture/environments/" + parts[1]}}
				scope.Identities = map[string]urn.V1{parts[1]: {WorkspaceID: "ws_fixture", Resource: "projects/proj_fixture/identities/" + parts[1]}}
			}
			got, err := migration.Translate(tt.input, scope)
			if tt.output == "" {
				require.ErrorIs(t, err, migration.ErrUnsupported)
				require.Empty(t, got)
				unsupported++
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.output, got)
			translated++
		})
	}
	t.Logf("translated=%d unsupported=%d", translated, unsupported)
}
