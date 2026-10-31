package gdprerasure

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
	columns []string
}

var (
	mysqlTablePattern      = regexp.MustCompile("(?s)CREATE TABLE `(\\w+)` \\((.*?)\\n\\);")
	clickhouseTablePattern = regexp.MustCompile("(?s)CREATE TABLE (?:IF NOT EXISTS )?(?:`?default`?\\.)?`?(\\w+)`?\\s*\\((.*?)\\n\\)")
	columnPattern          = regexp.MustCompile("(?m)^\\s*`?(\\w+)`?\\s+[A-Za-z]")
	mysqlStatementPattern  = regexp.MustCompile("(?m)^(DELETE FROM|UPDATE) `?(\\w+)`?$")
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
// table and that MySQL steps are scoped to their subject. MySQL steps run
// against a recorder, so the check covers the statement each step actually
// sends. pkg/clickhouse checks its own erasure statements.
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
					requireScopedStatement(t, s.name(), "DELETE FROM", s.table, s.query, registry.subject)
				case mysqlUpdate:
					require.Contains(t, mysqlSchema, s.table)
					requireScopedStatement(t, s.name(), "UPDATE", s.table, s.query, registry.subject)
				case clickhouseWorkspaceDelete:
					require.Contains(t, clickhouseSchema, s.table)
				case clickhouseEndUserDelete:
					require.Contains(t, clickhouseSchema, s.table)
				}
			}
		})
	}
}

// requireScopedStatement runs one batch of query against a recorder and
// requires that it writes table, is limited to batchRowsMax rows, binds only
// the subject's values, and refuses a subject without them.
func requireScopedStatement(t *testing.T, name, verb, table string, query mysqlQuery, s subject) {
	t.Helper()

	recorder := &recordingDB{DBTX: nil, statement: "", args: nil}
	_, err := query(context.Background(), db.NewQueries(recorder), s)
	require.NoError(t, err, name)

	target := mysqlStatementPattern.FindStringSubmatch(recorder.statement)
	require.NotNil(t, target, "%s sent no DELETE or UPDATE:\n%s", name, recorder.statement)
	require.Equal(t, verb, target[1], name)
	require.Equal(t, table, target[2], name)
	require.Regexp(t, fmt.Sprintf("(?m)^LIMIT %d$", batchRowsMax), recorder.statement, name)

	require.NotEmpty(t, recorder.args, "%s is not scoped to the subject", name)
	values := []string{s.workspaceID, s.externalID, s.userID}
	for _, arg := range recorder.args {
		require.Contains(t, values, boundString(t, arg), "%s binds a value the subject does not provide", name)
	}

	_, err = query(context.Background(), db.NewQueries(&recordingDB{DBTX: nil, statement: "", args: nil}), subject{})
	require.Error(t, err, "%s runs without the subject's ids", name)
}

func boundString(t *testing.T, arg any) string {
	t.Helper()
	switch value := arg.(type) {
	case string:
		return value
	case sql.NullString:
		require.True(t, value.Valid)
		return value.String
	case json.RawMessage:
		var decoded string
		require.NoError(t, json.Unmarshal(value, &decoded))
		return decoded
	default:
		t.Fatalf("unexpected argument type %T", arg)
		return ""
	}
}

// recordingDB captures the statement a generated query sends instead of
// running it.
type recordingDB struct {
	db.DBTX
	statement string
	args      []any
}

func (r *recordingDB) ExecContext(_ context.Context, statement string, args ...any) (sql.Result, error) {
	r.statement = statement
	r.args = args
	return driver.RowsAffected(0), nil
}

func registryTables(steps []step) (mysqlTables []string, clickhouseTables []string) {
	for _, s := range steps {
		switch s := s.(type) {
		case mysqlDelete:
			mysqlTables = append(mysqlTables, s.table)
		case clickhouseWorkspaceDelete:
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
			tables[match[1]] = schemaTable{columns: columns}
		}
	}
	require.NotEmpty(t, tables, "no tables parsed under %s", dir)
	return tables
}
