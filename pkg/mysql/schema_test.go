package mysql

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const caseSensitiveCollation = "COLLATE utf8mb4_0900_as_cs"

func TestSchemaVarcharCollations(t *testing.T) {
	entries, err := os.ReadDir("schema")
	require.NoError(t, err)

	caseSensitiveCount := 0
	inheritedCount := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		path := filepath.Join("schema", entry.Name())
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)

		for lineNumber, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(strings.ToLower(line), "varchar(") {
				continue
			}

			collationCount := strings.Count(strings.ToUpper(line), " COLLATE ")
			require.LessOrEqual(t, collationCount, 1, "%s:%d", path, lineNumber+1)

			if collationCount == 0 {
				inheritedCount++
				continue
			}

			caseSensitiveCount++
			require.Equal(t, 1, strings.Count(line, caseSensitiveCollation), "%s:%d", path, lineNumber+1)
		}
	}

	require.Positive(t, caseSensitiveCount, "schema must contain case-sensitive VARCHAR columns")
	require.Positive(t, inheritedCount, "schema must leave non-compared VARCHAR columns uncollated")
}

func TestComparedVarcharsAreCaseSensitive(t *testing.T) {
	comparedColumns := map[string][]string{
		"acme_challenges.sql":   {"domain_id", "workspace_id", "token", "authorization"},
		"apps.sql":              {"id", "workspace_id", "project_id", "default_branch"},
		"clickhouse_outbox.sql": {"workspace_id", "event_id"},
		"deployment_changes.sql": {
			"resource_id", "region_id",
		},
		"deployments.sql": {"id", "workspace_id", "project_id", "git_branch"},
		"identities.sql":  {"id", "external_id", "workspace_id", "environment"},
		"keys.sql":        {"id", "key_auth_id", "hash", "workspace_id", "owner_id"},
		"permissions.sql": {"id", "workspace_id", "name"},
		"ratelimit_overrides.sql": {
			"id", "workspace_id", "namespace_id", "identifier",
		},
		"roles.sql": {"id", "workspace_id", "name"},
	}

	for file, columns := range comparedColumns {
		data, err := os.ReadFile(filepath.Join("schema", file))
		require.NoError(t, err)

		ddl := string(data)
		for _, column := range columns {
			require.Contains(t, ddl, "`"+column+"` varchar(", "%s.%s must be VARCHAR", file, column)

			lineStart := strings.Index(ddl, "`"+column+"` varchar(")
			lineEnd := strings.Index(ddl[lineStart:], "\n")
			require.NotEqual(t, -1, lineEnd, "%s.%s must end with a newline", file, column)
			require.Contains(t, ddl[lineStart:lineStart+lineEnd], caseSensitiveCollation, "%s.%s", file, column)
		}
	}
}
