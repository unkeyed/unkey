package db_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/vaultreencryption/internal/db"
)

func TestUpdateVaultEncryptedKeyPreservesRowOwnershipAndDeletions(t *testing.T) {
	database, err := mysql.NewReplica(containers.MySQLIsolated(t).DSN, "rw", sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	queries := db.NewQueries(database)
	ctx := t.Context()
	keyrings := []string{uid.New(uid.WorkspacePrefix), uid.New(uid.WorkspacePrefix)}
	keyIDOld := uid.New(uid.TestPrefix)
	keys := make([]string, 100)
	updates := make([]db.UpdateVaultEncryptedKeyParams, 0, len(keys))
	for i := range keys {
		keys[i] = uid.New(uid.KeyPrefix)
		keyring := keyrings[i%len(keyrings)]
		require.NoError(t, pkgdb.Query.InsertKeyEncryption(ctx, database, pkgdb.InsertKeyEncryptionParams{
			WorkspaceID:     keyring,
			KeyID:           keys[i],
			Encrypted:       "before",
			EncryptionKeyID: keyIDOld,
		}))
		stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(ctx, database, keys[i])
		require.NoError(t, err)
		updates = append(updates, db.UpdateVaultEncryptedKeyParams{
			Pk:         stored.Pk,
			Keyring:    keyring,
			Ciphertext: fmt.Sprintf("ciphertext-%d'\\value", i),
			KeyID:      uid.New(uid.TestPrefix),
		})
	}
	require.NoError(t, pkgdb.Query.InsertKey(ctx, database, pkgdb.InsertKeyParams{
		ID:          keys[37],
		WorkspaceID: updates[37].Keyring,
		KeySpaceID:  uid.New(uid.KeySpacePrefix),
		Hash:        keys[37],
	}))
	require.NoError(t, pkgdb.Query.DeleteKeyByID(ctx, database, keys[37]))
	updates[63].Keyring = keyrings[0]

	var changed int64
	for _, update := range updates {
		affected, err := queries.UpdateVaultEncryptedKey(ctx, update)
		require.NoError(t, err)
		changed += affected
	}
	require.EqualValues(t, 98, changed)
	for i, key := range keys {
		stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(ctx, database, key)
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

	invalid := updates[0]
	invalid.Ciphertext = strings.Repeat("x", 1025)
	invalid.KeyID = uid.New(uid.TestPrefix)
	_, err = queries.UpdateVaultEncryptedKey(ctx, invalid)
	require.Error(t, err)
	stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(ctx, database, keys[0])
	require.NoError(t, err)
	require.Equal(t, updates[0].Ciphertext, stored.Encrypted, "a failed statement must preserve the row")
	require.Equal(t, updates[0].KeyID, stored.EncryptionKeyID)
}
