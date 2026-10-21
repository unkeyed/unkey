package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
)

func TestCreateRootKeyUsesNewStoreWithoutLegacyResources(t *testing.T) {
	h := NewHarness(t)
	workspace := h.Resources().UserWorkspace

	for _, table := range []string{
		"projects",
		"key_auth",
		"apis",
	} {
		var count int
		require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM "+table+" WHERE workspace_id = ?", workspace.ID).Scan(&count))
		require.Zero(t, count, table)
	}

	permission := "unkey:v1:" + workspace.ID + ":**#read"
	key := h.CreateRootKey(workspace.ID, permission, permission)
	keyID := h.RootKeyID(key)

	rootKey, err := db.Query.FindUnkeyRootKeyByID(t.Context(), h.DB.RO(), keyID)
	require.NoError(t, err)
	require.Equal(t, workspace.ID, rootKey.WorkspaceID)

	permissions, err := db.Query.ListUnkeyPermissionsByPrincipal(t.Context(), h.DB.RO(), db.ListUnkeyPermissionsByPrincipalParams{
		WorkspaceID:   workspace.ID,
		PrincipalType: db.UnkeyPrincipalPermissionsPrincipalTypeRootKey,
		PrincipalID:   keyID,
	})
	require.NoError(t, err)
	require.Equal(t, []string{permission}, permissions)

	var legacyKeys int
	require.NoError(t, h.DB.RO().QueryRowContext(t.Context(), "SELECT COUNT(*) FROM `keys` WHERE hash = ?", hash.Sha256(key)).Scan(&legacyKeys))
	require.Zero(t, legacyKeys)
}
