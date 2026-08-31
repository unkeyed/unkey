package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

// TestFindApisByKeyAuthIds covers the keyspace-to-api reverse mapping stage 2
// depends on to express its api-scoped requirements.
func TestFindApisByKeyAuthIds(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	workspace := h.Resources().UserWorkspace
	otherWorkspace := h.CreateWorkspace()

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	require.True(t, api.KeyAuthID.Valid)

	otherAPI := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   otherWorkspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	require.True(t, otherAPI.KeyAuthID.Valid)

	rows, err := db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{api.KeyAuthID.String},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, api.KeyAuthID.String, rows[0].KeyAuthID)
	require.Equal(t, api.ID, rows[0].ApiID)

	// Workspace-scoped: another workspace's keyspace resolves to nothing.
	rows, err = db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{otherAPI.KeyAuthID.String},
	})
	require.NoError(t, err)
	require.Empty(t, rows)

	// Unknown keyspace ids are simply absent from the result.
	rows, err = db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{api.KeyAuthID.String, "ks_does_not_exist"},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, api.ID, rows[0].ApiID)
}

// TestFindKeyAuthsByIdsAndWorkspaceReportsEncryption covers the column added to
// the keyspace batch query so stage 2 can decide whether the encrypt-key arm
// applies.
func TestFindKeyAuthsByIdsAndWorkspaceReportsEncryption(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	workspace := h.Resources().UserWorkspace

	plain := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})
	encrypted := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: true,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	rows, err := db.Query.FindKeyAuthsByIdsAndWorkspace(ctx, h.DB.RO(), db.FindKeyAuthsByIdsAndWorkspaceParams{
		WorkspaceID: workspace.ID,
		KeyAuthIds:  []string{plain.KeyAuthID.String, encrypted.KeyAuthID.String},
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)

	byID := make(map[string]bool, len(rows))
	for _, row := range rows {
		byID[row.ID] = row.StoreEncryptedKeys
	}
	require.False(t, byID[plain.KeyAuthID.String])
	require.True(t, byID[encrypted.KeyAuthID.String])
}
