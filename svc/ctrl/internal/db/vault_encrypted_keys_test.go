package db_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestBulkUpdateVaultEncryptedKeysPreservesRowOwnershipAndDeletions(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	ctx := t.Context()
	keyrings := []string{uid.New(uid.WorkspacePrefix), uid.New(uid.WorkspacePrefix)}
	keyIDOld := uid.New(uid.TestPrefix)
	keys := make([]string, 100)
	updates := make([]db.UpdateVaultEncryptedKeyParams, 0, len(keys))
	for i := range keys {
		keys[i] = uid.New(uid.KeyPrefix)
		keyring := keyrings[i%len(keyrings)]
		require.NoError(t, pkgdb.Query.InsertKeyEncryption(ctx, database.RW(), pkgdb.InsertKeyEncryptionParams{
			WorkspaceID:     keyring,
			KeyID:           keys[i],
			Encrypted:       "before",
			EncryptionKeyID: keyIDOld,
		}))
		stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(ctx, database.RW(), keys[i])
		require.NoError(t, err)
		updates = append(updates, db.UpdateVaultEncryptedKeyParams{
			Pk:         stored.Pk,
			Keyring:    keyring,
			Ciphertext: fmt.Sprintf("ciphertext-%d'\\value", i),
			KeyID:      uid.New(uid.TestPrefix),
		})
	}
	require.NoError(t, pkgdb.Query.InsertKey(ctx, database.RW(), pkgdb.InsertKeyParams{
		ID:          keys[37],
		WorkspaceID: updates[37].Keyring,
		KeySpaceID:  uid.New(uid.KeySpacePrefix),
		Hash:        keys[37],
	}))
	require.NoError(t, pkgdb.Query.DeleteKeyByID(ctx, database.RW(), keys[37]))
	updates[63].Keyring = keyrings[0]

	changed, err := database.Bulk().UpdateVaultEncryptedKeys(ctx, updates)
	require.NoError(t, err)
	require.EqualValues(t, 98, changed)
	for i, key := range keys {
		stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(ctx, database.RW(), key)
		if i == 37 {
			require.ErrorIs(t, err, sql.ErrNoRows)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, keyrings[i%len(keyrings)], stored.WorkspaceID)
		if i == 63 {
			require.Equal(t, "before", stored.Encrypted)
			require.Equal(t, keyIDOld, stored.EncryptionKeyID)
			continue
		}
		require.Equal(t, updates[i].Ciphertext, stored.Encrypted)
		require.Equal(t, updates[i].KeyID, stored.EncryptionKeyID)
	}

	invalid := []db.UpdateVaultEncryptedKeyParams{updates[0], updates[99]}
	invalid[0].Ciphertext = "must not persist"
	invalid[1].Ciphertext = strings.Repeat("x", 1025)
	_, err = database.Bulk().UpdateVaultEncryptedKeys(ctx, invalid)
	require.Error(t, err)
	stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(ctx, database.RW(), keys[0])
	require.NoError(t, err)
	require.Equal(t, updates[0].Ciphertext, stored.Encrypted, "a failed statement must not write earlier rows")

	changed, err = database.Bulk().UpdateVaultEncryptedKeys(ctx, nil)
	require.NoError(t, err)
	require.Zero(t, changed)
}
