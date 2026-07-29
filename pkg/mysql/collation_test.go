package mysql

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

// TestStagedCollationBridge_JoinsAcrossPartialMigration guarantees that ID
// joins remain case-sensitive while counterpart columns migrate independently.
// The native-collation predicate preserves index eligibility, while the as_cs
// predicate rejects case-insensitive matches in every rollout state.
func TestStagedCollationBridge_JoinsAcrossPartialMigration(t *testing.T) {
	testCases := []struct {
		name                     string
		parentCollation          string
		childCollation           string
		plainJoinMatchesExpected int
	}{
		{
			name:                     "both columns are case-insensitive",
			parentCollation:          "utf8mb4_0900_ai_ci",
			childCollation:           "utf8mb4_0900_ai_ci",
			plainJoinMatchesExpected: 2,
		},
		{
			name:                     "parent migrated before child",
			parentCollation:          "utf8mb4_0900_as_cs",
			childCollation:           "utf8mb4_0900_ai_ci",
			plainJoinMatchesExpected: -1,
		},
		{
			name:                     "child migrated before parent",
			parentCollation:          "utf8mb4_0900_ai_ci",
			childCollation:           "utf8mb4_0900_as_cs",
			plainJoinMatchesExpected: -1,
		},
		{
			name:                     "both columns are case-sensitive",
			parentCollation:          "utf8mb4_0900_as_cs",
			childCollation:           "utf8mb4_0900_as_cs",
			plainJoinMatchesExpected: 1,
		},
	}

	config := containers.MySQL(t)
	db, err := sql.Open("mysql", config.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	connection, err := db.Conn(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err = connection.ExecContext(t.Context(), "DROP TEMPORARY TABLE IF EXISTS collation_parent, collation_child")
			require.NoError(t, err)

			_, err = connection.ExecContext(t.Context(), fmt.Sprintf(`
				CREATE TEMPORARY TABLE collation_parent (
					id varchar(64) COLLATE %s NOT NULL PRIMARY KEY
				);
				CREATE TEMPORARY TABLE collation_child (
					parent_id varchar(64) COLLATE %s NOT NULL,
					INDEX parent_id_idx (parent_id)
				);
			`, testCase.parentCollation, testCase.childCollation))
			require.NoError(t, err)

			_, err = connection.ExecContext(t.Context(), `
				INSERT INTO collation_parent (id) VALUES ('CaseID');
				INSERT INTO collation_child (parent_id) VALUES ('CaseID'), ('caseid');
			`)
			require.NoError(t, err)

			var plainJoinMatches int
			err = connection.QueryRowContext(t.Context(), `
				SELECT COUNT(*)
				FROM collation_parent parent
				JOIN collation_child child ON child.parent_id = parent.id
			`).Scan(&plainJoinMatches)
			if testCase.plainJoinMatchesExpected >= 0 {
				require.NoError(t, err)
				require.Equal(t, testCase.plainJoinMatchesExpected, plainJoinMatches)
			} else {
				require.Error(t, err, "mixed implicit collations must demonstrate the rollout failure")
			}

			var bridgedJoinMatches int
			err = connection.QueryRowContext(t.Context(), `
				SELECT COUNT(*)
				FROM collation_parent parent
				JOIN collation_child child ON (
					child.parent_id = parent.id COLLATE utf8mb4_0900_ai_ci
					AND child.parent_id = parent.id COLLATE utf8mb4_0900_as_cs
				)
			`).Scan(&bridgedJoinMatches)
			require.NoError(t, err)
			require.Equal(t, 1, bridgedJoinMatches)
		})
	}
}
