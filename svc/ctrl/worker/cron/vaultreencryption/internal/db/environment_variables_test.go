package db_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/vaultreencryption/internal/db"
)

func TestUpdateEnvironmentVariableRequiresExactCiphertext(t *testing.T) {
	database, err := mysql.NewReplica(containers.MySQLIsolated(t).DSN, "rw", sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	queries := db.NewQueries(database)
	ctx := t.Context()
	keyring := uid.New(uid.EnvironmentPrefix)
	require.NoError(t, pkgdb.Query.InsertAppEnvironmentVariable(ctx, database, pkgdb.InsertAppEnvironmentVariableParams{
		ID:            uid.New(uid.TestPrefix),
		WorkspaceID:   uid.New(uid.WorkspacePrefix),
		AppID:         uid.New(uid.AppPrefix),
		EnvironmentID: keyring,
		EnvKey:        "TOKEN",
		Type:          pkgdb.AppEnvironmentVariablesTypeRecoverable,
		Value:         "QXNk",
	}))
	rows, err := queries.ListEnvironmentVariables(ctx, db.ListEnvironmentVariablesParams{Limit: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	for _, ciphertextOld := range []string{"qxnk", "QXNk "} {
		t.Run(ciphertextOld, func(t *testing.T) {
			updated, err := queries.UpdateEnvironmentVariable(ctx, db.UpdateEnvironmentVariableParams{
				Pk:            rows[0].Pk,
				Keyring:       keyring,
				CiphertextOld: ciphertextOld,
				Ciphertext:    "replacement",
			})
			require.NoError(t, err)
			require.Zero(t, updated, "a byte-different ciphertext must conflict")
		})
	}
	rows, err = queries.ListEnvironmentVariables(ctx, db.ListEnvironmentVariablesParams{Limit: 1})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "QXNk", rows[0].Ciphertext)
	updated, err := queries.UpdateEnvironmentVariable(ctx, db.UpdateEnvironmentVariableParams{
		Pk:            rows[0].Pk,
		Keyring:       keyring,
		CiphertextOld: "QXNk",
		Ciphertext:    "replacement",
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)
}
