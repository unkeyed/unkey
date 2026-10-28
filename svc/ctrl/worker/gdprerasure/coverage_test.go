package gdprerasure

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// retainedMySQLTables are schema tables a workspace erasure deliberately leaves
// alone, each with the reason.
var retainedMySQLTables = map[string]string{
	"clusters": "global infrastructure inventory, holds no workspace data",
	"flags":    "global feature flag definitions; per-workspace overrides live in flag_overrides",
	"regions":  "global infrastructure inventory, holds no workspace data",
}

// retainedClickHouseTables are ClickHouse tables with a workspace_id that a
// workspace erasure deliberately leaves alone, each with the reason.
var retainedClickHouseTables = map[string]string{}

type schemaTable struct {
	columns       []string
	hasProjection bool
}

var (
	mysqlTablePattern      = regexp.MustCompile("(?s)CREATE TABLE `(\\w+)` \\((.*?)\\n\\);")
	clickhouseTablePattern = regexp.MustCompile("(?s)CREATE TABLE (?:IF NOT EXISTS )?(?:`?default`?\\.)?`?(\\w+)`?\\s*\\((.*?)\\n\\)")
	columnPattern          = regexp.MustCompile("(?m)^\\s*`?(\\w+)`?\\s+[A-Za-z]")
	projectionPattern      = regexp.MustCompile(`(?m)^\s*PROJECTION\s`)
)

// TestWorkspaceErasureCoversEverySchemaTable guarantees that a new table cannot
// silently keep a workspace's personal data after a GDPR erasure. Every MySQL
// table must be erased by the workspace registry or retained with a reason,
// and so must every ClickHouse table that has a workspace_id.
func TestWorkspaceErasureCoversEverySchemaTable(t *testing.T) {
	erasedMySQL, erasedClickHouse := registryTables(workspaceSteps)

	t.Run("mysql", func(t *testing.T) {
		var uncovered []string
		for table := range readSchema(t, "../../../../pkg/mysql/schema", mysqlTablePattern) {
			if !accounted(t, table, erasedMySQL, retainedMySQLTables) {
				uncovered = append(uncovered, table)
			}
		}
		slices.Sort(uncovered)
		require.Empty(t, uncovered, "MySQL tables neither erased by workspaceSteps nor retained with a reason")
	})

	t.Run("clickhouse", func(t *testing.T) {
		var uncovered []string
		for table, definition := range readSchema(t, "../../../../pkg/clickhouse/schema", clickhouseTablePattern) {
			if slices.Contains(definition.columns, "workspace_id") && !accounted(t, table, erasedClickHouse, retainedClickHouseTables) {
				uncovered = append(uncovered, table)
			}
		}
		slices.Sort(uncovered)
		require.Empty(t, uncovered, "ClickHouse tables with a workspace_id neither erased by workspaceSteps nor retained with a reason")
	})
}

func accounted(t *testing.T, table string, erased []string, retained map[string]string) bool {
	t.Helper()
	reason, isRetained := retained[table]
	if !isRetained {
		return slices.Contains(erased, table)
	}
	require.NotEmpty(t, reason, "retained table %s needs a reason", table)
	require.NotContains(t, erased, table, "table %s is both erased and retained", table)
	return true
}

// TestRegistriesMatchSchema guarantees that every registry step names a real
// table and binds only placeholders its subject provides, and that ClickHouse
// tables with a projection are deleted with a mutation, because they reject a
// lightweight DELETE outright.
func TestRegistriesMatchSchema(t *testing.T) {
	mysqlSchema := readSchema(t, "../../../../pkg/mysql/schema", mysqlTablePattern)
	clickhouseSchema := readSchema(t, "../../../../pkg/clickhouse/schema", clickhouseTablePattern)

	registries := []struct {
		name    string
		steps   []step
		subject subject
	}{
		{name: "workspace", steps: workspaceSteps, subject: subject{workspaceID: "w", externalID: "", userID: "", identityIDs: nil, orgID: "", stripeCustomerID: ""}},
		{name: "end user", steps: endUserSteps, subject: subject{workspaceID: "w", externalID: "e", userID: "", identityIDs: []string{"i"}, orgID: "", stripeCustomerID: ""}},
		{name: "user", steps: userSteps, subject: subject{workspaceID: "", externalID: "", userID: "u", identityIDs: nil, orgID: "", stripeCustomerID: ""}},
	}

	for _, registry := range registries {
		t.Run(registry.name, func(t *testing.T) {
			names := make(map[string]bool, len(registry.steps))
			for _, s := range registry.steps {
				require.False(t, names[s.name()], "step name %q repeats", s.name())
				names[s.name()] = true

				switch s := s.(type) {
				case mysqlDelete:
					require.Contains(t, mysqlSchema, s.table)
					_, _, err := registry.subject.bind(s.where)
					require.NoError(t, err, s.name())
				case mysqlUpdate:
					require.Contains(t, mysqlSchema, s.table)
					_, _, err := registry.subject.bind(s.set + " " + s.where)
					require.NoError(t, err, s.name())
				case clickhouseDelete:
					require.Contains(t, clickhouseSchema, s.table)
					_, _, err := registry.subject.bind(s.where)
					require.NoError(t, err, s.name())
					if clickhouseSchema[s.table].hasProjection {
						require.Equal(t, deleteMutation, s.mode, "%s has a projection and rejects lightweight deletes", s.table)
					}
				case clickhouseUpdate:
					require.Contains(t, clickhouseSchema, s.table)
					_, _, err := registry.subject.bind(s.set + " " + s.where)
					require.NoError(t, err, s.name())
				}
			}
		})
	}
}

func registryTables(steps []step) (mysqlTables []string, clickhouseTables []string) {
	for _, s := range steps {
		switch s := s.(type) {
		case mysqlDelete:
			mysqlTables = append(mysqlTables, s.table)
		case clickhouseDelete:
			clickhouseTables = append(clickhouseTables, s.table)
		}
	}
	return mysqlTables, clickhouseTables
}

func readSchema(t *testing.T, dir string, tablePattern *regexp.Regexp) map[string]schemaTable {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "no schema files under %s", dir)

	tables := make(map[string]schemaTable)
	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)
		for _, match := range tablePattern.FindAllStringSubmatch(string(content), -1) {
			var columns []string
			for _, column := range columnPattern.FindAllStringSubmatch(match[2], -1) {
				columns = append(columns, column[1])
			}
			tables[match[1]] = schemaTable{columns: columns, hasProjection: projectionPattern.MatchString(match[2])}
		}
	}
	require.NotEmpty(t, tables, "no tables parsed under %s", dir)
	return tables
}
