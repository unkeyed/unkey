package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_BlockNonWhitelistedFunctions(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications_v1": "default.key_verifications_raw_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name       string
		query      string
		shouldFail bool
	}{
		{
			name:       "file function",
			query:      "SELECT file('/etc/passwd') FROM key_verifications_v1",
			shouldFail: true,
		},
		{
			name:       "url function",
			query:      "SELECT url('http://evil.com/data') FROM key_verifications_v1",
			shouldFail: true,
		},
		{
			name:       "system function",
			query:      "SELECT system('rm -rf /') FROM key_verifications_v1",
			shouldFail: true,
		},
		{
			name:       "executable function",
			query:      "SELECT executable('/bin/bash') FROM key_verifications_v1",
			shouldFail: true,
		},
		{
			name:       "dict_get to access dictionaries",
			query:      "SELECT dictGet('dict', 'attr', key_id) FROM key_verifications_v1",
			shouldFail: true,
		},
		{
			name:       "nested safe functions should work",
			query:      "SELECT count(DISTINCT key_id) FROM key_verifications_v1",
			shouldFail: false,
		},
		{
			// The runtime logs endpoint reads attributes out of the
			// attributes_text JSON string. Only this member of the JSONExtract
			// group has a grant.
			name:       "json extract on a string column",
			query:      "SELECT JSONExtractString(key_id, 'user_id') FROM key_verifications_v1",
			shouldFail: false,
		},
		{
			name:       "json extract does not admit its arguments",
			query:      "SELECT JSONExtractString(file('/etc/passwd'), 'user_id') FROM key_verifications_v1",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)

			if tt.shouldFail {
				require.Error(t, err, "Dangerous function should be blocked")
				require.Contains(t, err.Error(), "not allowed", "Error should indicate function is not allowed")
			} else {
				require.NoError(t, err, "Safe function combination should work")
			}
		})
	}
}

func TestParser_AllowSafeFunctions(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"keys_v1": "default.keys_v2",
		},
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	safeFuncs := []string{"count", "sum", "avg", "max", "min", "now", "toDate"}

	for _, fn := range safeFuncs {
		query := "SELECT " + fn + "(*) FROM keys_v1"
		_, err := p.Parse(context.Background(), query)
		require.NoError(t, err, "Function %s should be allowed", fn)
	}
}

func TestParser_OnlySelectAllowed(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "INSERT statement",
			query: "INSERT INTO default.key_verifications_raw_v2 (key_id) VALUES ('malicious')",
		},
		{
			name:  "UPDATE statement",
			query: "UPDATE default.key_verifications_raw_v2 SET key_id = 'hacked'",
		},
		{
			name:  "DELETE statement",
			query: "DELETE FROM default.key_verifications_raw_v2",
		},
		{
			name:  "DROP statement",
			query: "DROP TABLE default.key_verifications_raw_v2",
		},
		{
			name:  "CREATE statement",
			query: "CREATE TABLE malicious (id INT)",
		},
		{
			name:  "ALTER statement",
			query: "ALTER TABLE default.key_verifications_raw_v2 ADD COLUMN backdoor STRING",
		},
		{
			name:  "TRUNCATE statement",
			query: "TRUNCATE TABLE default.key_verifications_raw_v2",
		},
		{
			name:  "GRANT statement",
			query: "GRANT ALL ON *.* TO 'attacker'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			require.Error(t, err, "Only SELECT queries should be allowed")
		})
	}
}

// TestParser_RejectsIntrospectionStatements guarantees schema introspection
// never gets to ClickHouse. These statements list every column of a table
// without selecting from it, so a column allow-list on a SELECT does not apply
// to them. ClickHouse also refuses them for a workspace user, but the parser is
// the first of the two barriers.
func TestParser_RejectsIntrospectionStatements(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"events_v1": "default.events",
		},
		AllowedTables: []string{"default.events"},
	})

	queries := []string{
		"DESCRIBE TABLE events_v1",
		"DESCRIBE events_v1",
		"DESC events_v1",
		"SHOW CREATE TABLE events_v1",
		"SHOW COLUMNS FROM events_v1",
		"SHOW TABLES",
		"EXISTS TABLE events_v1",
		"EXPLAIN SELECT count() FROM events_v1",
		"SELECT count() FROM events_v1; DESCRIBE TABLE events_v1",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), query)
			require.Error(t, err, "introspection must not reach ClickHouse")
		})
	}
}

// TestParser_RejectsMultipleStatements guarantees validation covers the full
// input instead of executing a valid first statement and ignoring the rest.
func TestParser_RejectsMultipleStatements(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"events_v1": "default.events",
		},
		AllowedTables: []string{"default.events"},
	})

	output, err := parser.Parse(context.Background(), "SELECT count() FROM events_v1;")
	require.NoError(t, err)
	require.NotEmpty(t, output)

	queries := []struct {
		name  string
		query string
	}{
		{name: "second select", query: "SELECT count() FROM events_v1; SELECT count() FROM system.tables"},
		{name: "second non-select", query: "SELECT count() FROM events_v1; DROP TABLE default.events"},
		{name: "physical table in second statement", query: "SELECT count() FROM events_v1; SELECT count() FROM default.events"},
	}
	for _, test := range queries {
		t.Run(test.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), test.query)
			require.Error(t, err)
		})
	}
}

// TestParser_EmitsClickHouseCompatibleAllowedFunctionNames guarantees an
// allowlisted function cannot pass validation and then fail only due to casing.
func TestParser_EmitsClickHouseCompatibleAllowedFunctionNames(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"events_v1": "default.events",
		},
		AllowedTables: []string{"default.events"},
	})

	output, err := parser.Parse(context.Background(), "SELECT COUNTIF(id = 'evt_123') FROM events_v1")
	require.NoError(t, err)
	require.Contains(t, output, "countIf(")
	require.NotContains(t, output, "COUNTIF(")

	output, err = parser.Parse(context.Background(), "SELECT jsonextractstring(attributes_text, 'user_id') FROM events_v1")
	require.NoError(t, err)
	require.Contains(t, output, "JSONExtractString(")
	require.NotContains(t, output, "jsonextractstring(")
}

// TestParser_RejectsSettingsClauses guarantees customer SQL cannot attach
// query-level settings to the root query or any nested SELECT query.
func TestParser_RejectsSettingsClauses(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"keys_v1": "default.keys_v2",
		},
		AllowedTables: []string{"default.keys_v2"},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "top level",
			query: "SELECT * FROM keys_v1 SETTINGS max_execution_time = 0",
		},
		{
			name:  "scalar subquery",
			query: "SELECT (SELECT count(*) FROM keys_v1 SETTINGS max_execution_time = 0) FROM keys_v1",
		},
		{
			name:  "from subquery",
			query: "SELECT * FROM (SELECT * FROM keys_v1 SETTINGS max_execution_time = 0)",
		},
		{
			name:  "common table expression",
			query: "WITH cte AS (SELECT * FROM keys_v1 SETTINGS max_execution_time = 0) SELECT * FROM cte",
		},
		{
			name:  "union branch",
			query: "SELECT * FROM keys_v1 UNION ALL SELECT * FROM keys_v1 SETTINGS max_execution_time = 0",
		},
		{
			name:  "except branch",
			query: "SELECT * FROM keys_v1 EXCEPT SELECT * FROM keys_v1 SETTINGS max_execution_time = 0",
		},
		{
			name:  "deeply nested",
			query: "SELECT * FROM (SELECT * FROM (SELECT * FROM keys_v1 SETTINGS max_execution_time = 0))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			require.ErrorContains(t, err, "SETTINGS clauses are not allowed")
		})
	}
}

// TestParser_RejectsTableBackedSetOperands guarantees a secondary table read
// cannot bypass public aliases or row-level filters through ClickHouse's
// unparenthesized IN-table syntax, including from an EXCEPT branch.
func TestParser_RejectsTableBackedSetOperands(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"events_v1": "default.events",
		},
		AllowedTables: []string{"default.events"},
	})

	queries := []string{
		"SELECT * FROM events_v1 WHERE id IN default.events",
		"SELECT * FROM events_v1 WHERE id NOT IN events_v1",
		"SELECT * FROM events_v1 WHERE id GLOBAL IN default.events",
		"SELECT * FROM events_v1 WHERE id IN numbers(10)",
		"SELECT * FROM events_v1 WHERE id IN {ids:Array(String)}",
		"SELECT * FROM events_v1 EXCEPT SELECT * FROM events_v1 WHERE id IN default.events",
	}
	for _, query := range queries {
		_, err := parser.Parse(context.Background(), query)
		require.ErrorContains(t, err, "table-backed set expressions are not supported", query)
	}
}

// TestParser_AllowsFilterableSetOperands guarantees ordinary literal sets and
// subqueries remain supported while every subquery source receives validation.
func TestParser_AllowsFilterableSetOperands(t *testing.T) {
	parser := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"events_v1": "default.events",
		},
		AllowedTables: []string{"default.events"},
	})

	queries := []string{
		"SELECT * FROM events_v1 WHERE id IN ('first', 'second')",
		"SELECT * FROM events_v1 WHERE id NOT IN ['first', 'second']",
		"SELECT * FROM events_v1 WHERE id IN (SELECT id FROM events_v1)",
	}
	for _, query := range queries {
		_, err := parser.Parse(context.Background(), query)
		require.NoError(t, err, query)
	}
}

// TestParser_ParametricAggregates covers the parametric aggregate form
// `f(0.95)(col)`. The vendored parser models it as a ParamExprList carrying a
// ColumnArgList, so this pins that the validation walk descends into both
// argument lists.
func TestParser_ParametricAggregates(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_KEBAP",
		TableAliases: map[string]string{
			"events_v1": "default.events",
		},
		AllowedTables: []string{
			"default.events",
		},
	})

	tests := []struct {
		name       string
		query      string
		shouldFail bool
	}{
		{
			name:       "allowed parametric aggregate",
			query:      "SELECT quantile(0.95)(latency) AS p95 FROM events_v1",
			shouldFail: false,
		},
		{
			// The parametric position must not become a hole in the function
			// allow-list. If this passes, the walk is not reaching ColumnArgList
			// and every disallowed function can be smuggled through it.
			name:       "disallowed function in the column argument",
			query:      "SELECT quantile(0.95)(file('/etc/passwd')) FROM events_v1",
			shouldFail: true,
		},
		{
			name:       "disallowed function in the parameter argument",
			query:      "SELECT quantile(file('/etc/passwd'))(latency) FROM events_v1",
			shouldFail: true,
		},
		{
			name:       "state accessor stays blocked",
			query:      "SELECT quantileTDigestState(0.95)(latency) FROM events_v1",
			shouldFail: true,
		},
		{
			name:       "merge accessor stays blocked",
			query:      "SELECT quantileTDigestMerge(0.95)(latency) FROM events_v1",
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			if tt.shouldFail {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
