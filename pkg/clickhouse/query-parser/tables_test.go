package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestParser_TableAliases(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"keys": "default.keys_v2",
		},
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM keys")
	require.NoError(t, err)

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE workspace_id = 'ws_123'", output)
}

func TestParser_BlockSystemTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "system.tables",
			query: "SELECT * FROM system.tables",
		},
		{
			name:  "system.columns",
			query: "SELECT * FROM system.columns",
		},
		{
			name:  "system.databases",
			query: "SELECT * FROM system.databases",
		},
		{
			name:  "system.users",
			query: "SELECT * FROM system.users",
		},
		{
			name:  "system.query_log",
			query: "SELECT * FROM system.query_log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			require.Error(t, err, "Access to system tables should be blocked")
			require.Contains(t, err.Error(), "not allowed")
		})
	}
}

func TestParser_AllowedTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	// Allowed table should work
	_, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2")
	require.NoError(t, err)

	// Non-allowed table should fail
	_, err = p.Parse(context.Background(), "SELECT * FROM default.other_table")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestParser_RejectsTableBackedMembershipOperands(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	for _, operator := range []string{"IN", "NOT IN", "GLOBAL IN", "GLOBAL NOT IN"} {
		for _, table := range []string{"ratelimits_raw_v2", "default.ratelimits_raw_v2"} {
			t.Run(operator+"_"+table, func(t *testing.T) {
				query := "SELECT key_id FROM default.key_verifications_raw_v2 WHERE key_id " + operator + " " + table
				_, err := p.Parse(context.Background(), query)

				// A physical table on an IN-family RHS must be rejected before ClickHouse can read it.
				require.Error(t, err)
				code, ok := fault.GetCode(err)
				require.True(t, ok)
				if operator == "GLOBAL NOT IN" {
					require.Equal(t, codes.User.BadRequest.InvalidAnalyticsQuery.URN(), code)
				} else {
					require.Equal(t, codes.User.BadRequest.InvalidAnalyticsTable.URN(), code)
					require.Contains(t, err.Error(), "right-hand operand not allowed")
				}
			})
		}
	}
}

func TestParser_TableFamiliesCannotCrossViaMembershipRHS(t *testing.T) {
	tests := map[string]struct {
		allowedTable string
		rhsTable     string
	}{
		"verifications cannot read ratelimits": {
			allowedTable: "default.key_verifications_raw_v2",
			rhsTable:     "default.ratelimits_raw_v2",
		},
		"ratelimits cannot read verifications": {
			allowedTable: "default.ratelimits_raw_v2",
			rhsTable:     "default.key_verifications_raw_v2",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := NewParser(Config{
				WorkspaceID:   "ws_123",
				AllowedTables: []string{test.allowedTable},
			})

			_, err := p.Parse(context.Background(), "SELECT workspace_id FROM "+test.allowedTable+" WHERE workspace_id IN "+test.rhsTable)

			// Each analytics route's physical table family must remain isolated.
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.User.BadRequest.InvalidAnalyticsTable.URN(), code)
		})
	}
}

func TestParser_RejectsTableBackedMembershipOperandsInEveryQueryShape(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := map[string]string{
		"nested": "SELECT * FROM (SELECT key_id FROM default.key_verifications_raw_v2 WHERE key_id IN default.ratelimits_raw_v2)",
		"CTE":    "WITH ids AS (SELECT key_id FROM default.key_verifications_raw_v2 WHERE key_id NOT IN default.ratelimits_raw_v2) SELECT * FROM ids",
		"UNION":  "SELECT key_id FROM default.key_verifications_raw_v2 UNION ALL SELECT key_id FROM default.key_verifications_raw_v2 WHERE key_id GLOBAL IN default.ratelimits_raw_v2",
	}

	for name, query := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), query)

			// Nesting, CTEs, and UNION branches must not hide a table-backed RHS.
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.User.BadRequest.InvalidAnalyticsTable.URN(), code)
		})
	}
}

func TestParser_RejectsDynamicMembershipOperands(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	for name, operand := range map[string]string{
		"identifier":               "other_ids",
		"path":                     "default.other_ids",
		"placeholder":              "?",
		"placeholder list":         "(?)",
		"parameter":                "{ids:Array(String)}",
		"backtick identifier":      "(`TRUE`)",
		"double quoted identifier": `("FALSE")`,
		"identifier in tuple":      "((`NULL`, 'VALID'))",
	} {
		t.Run(name, func(t *testing.T) {
			query := "SELECT key_id FROM default.key_verifications_raw_v2 WHERE key_id IN " + operand
			_, err := p.Parse(context.Background(), query)

			// Dynamic RHS operands must not defer table selection or authorization to execution time.
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, codes.User.BadRequest.InvalidAnalyticsTable.URN(), code)
		})
	}
}

func TestParser_PreservesLiteralMembershipLists(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := map[string]string{
		"strings":        "key_id IN ('key_1', 'key_2')",
		"numbers":        "key_id NOT IN (1, 2)",
		"signed numbers": "key_id GLOBAL IN (-1, 0, 1)",
		"booleans":       "key_id IN (TRUE, FALSE)",
		"null":           "key_id IN (NULL, 'key_1')",
		"tuples":         "(key_id, outcome) IN (('key_1', 'VALID'), ('key_2', 'INVALID'))",
	}

	for name, predicate := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := p.Parse(context.Background(), "SELECT key_id FROM default.key_verifications_raw_v2 WHERE "+predicate)

			// Literal lists are values, not table sources, and must remain supported.
			require.NoError(t, err)
			require.Contains(t, output, predicate)
		})
	}
}

func TestParser_RejectsMembershipSubqueries(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := map[string]struct {
		query        string
		expectedCode codes.URN
	}{
		"SELECT": {
			query:        "SELECT key_id FROM default.key_verifications_raw_v2",
			expectedCode: codes.User.BadRequest.InvalidAnalyticsTable.URN(),
		},
		"nested": {
			query:        "SELECT key_id FROM (SELECT key_id FROM default.key_verifications_raw_v2)",
			expectedCode: codes.User.BadRequest.InvalidAnalyticsTable.URN(),
		},
		"CTE": {
			query:        "WITH ids AS (SELECT key_id FROM default.key_verifications_raw_v2) SELECT key_id FROM ids",
			expectedCode: codes.User.BadRequest.InvalidAnalyticsQuery.URN(),
		},
		"UNION": {
			query:        "SELECT key_id FROM default.key_verifications_raw_v2 UNION ALL SELECT key_id FROM default.key_verifications_raw_v2",
			expectedCode: codes.User.BadRequest.InvalidAnalyticsTable.URN(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), "SELECT key_id FROM default.key_verifications_raw_v2 WHERE key_id IN ("+test.query+")")

			// Subquery RHS sources are rejected until each source can be independently filtered.
			require.Error(t, err)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, test.expectedCode, code)
			if code == codes.User.BadRequest.InvalidAnalyticsTable.URN() {
				require.Contains(t, err.Error(), "right-hand operand not allowed")
			}
		})
	}
}

func TestParser_BlockInformationSchema(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "information_schema.tables",
			query: "SELECT * FROM information_schema.tables",
		},
		{
			name:  "INFORMATION_SCHEMA.COLUMNS uppercase",
			query: "SELECT * FROM INFORMATION_SCHEMA.COLUMNS",
		},
		{
			name:  "information_schema.schemata",
			query: "SELECT * FROM information_schema.schemata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			require.Error(t, err, "Access to information_schema should be blocked")
			require.Contains(t, err.Error(), "not allowed")
		})
	}
}

func TestParser_UNIONWithTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       1000,
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name        string
		query       string
		shouldBlock bool
	}{
		{
			name:        "UNION DISTINCT to system tables",
			query:       "SELECT key_id FROM default.key_verifications_raw_v2 UNION DISTINCT SELECT name FROM system.tables",
			shouldBlock: true,
		},
		{
			name:        "UNION ALL to bypass deduplication",
			query:       "SELECT key_id FROM default.key_verifications_raw_v2 UNION ALL SELECT name FROM system.databases",
			shouldBlock: true,
		},
		{
			name:        "UNION DISTINCT with unauthorized table",
			query:       "SELECT key_id FROM default.key_verifications_raw_v2 UNION DISTINCT SELECT id FROM default.secrets",
			shouldBlock: true,
		},
		{
			name:        "UNION ALL with another allowed table",
			query:       "SELECT key_id FROM default.key_verifications_raw_v2 UNION ALL SELECT key_id FROM default.key_verifications_raw_v2",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)

			if tt.shouldBlock {
				require.Error(t, err, "UNION with unauthorized/system tables should be blocked")
				require.Contains(t, err.Error(), "not allowed", "Error should indicate table is not allowed")
			} else {
				require.NoError(t, err, "UNION with allowed tables should work")
			}
		})
	}
}

func TestParser_JOINWithTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name        string
		query       string
		shouldBlock bool
	}{
		{
			name:        "INNER JOIN to system table",
			query:       "SELECT * FROM default.key_verifications_raw_v2 INNER JOIN system.tables ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "LEFT JOIN to unauthorized table",
			query:       "SELECT * FROM default.key_verifications_raw_v2 LEFT JOIN default.secrets ON key_id = id",
			shouldBlock: true,
		},
		{
			name:        "RIGHT JOIN to system table",
			query:       "SELECT * FROM default.key_verifications_raw_v2 RIGHT JOIN system.databases ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "CROSS JOIN to system table",
			query:       "SELECT * FROM default.key_verifications_raw_v2 CROSS JOIN system.users",
			shouldBlock: true,
		},
		{
			name:        "Multiple JOINs with one unauthorized",
			query:       "SELECT * FROM default.key_verifications_raw_v2 t1 JOIN default.key_verifications_raw_v2 t2 ON t1.key_id = t2.key_id JOIN system.tables t3 ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "JOIN with allowed tables",
			query:       "SELECT * FROM default.key_verifications_raw_v2 t1 INNER JOIN default.key_verifications_raw_v2 t2 ON t1.key_id = t2.key_id",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)

			if tt.shouldBlock {
				require.Error(t, err, "JOIN with unauthorized/system tables should be blocked")
				require.Contains(t, err.Error(), "not allowed", "Error should indicate table is not allowed")
			} else {
				require.NoError(t, err, "JOIN with allowed tables should work")
			}
		})
	}
}

func TestParser_SubqueryWithTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       10,
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name        string
		query       string
		expected    string
		shouldBlock bool
	}{
		{
			name:        "subquery in FROM with allowed table",
			query:       "SELECT * FROM (SELECT * FROM default.key_verifications_raw_v2 LIMIT 10000) LIMIT 5",
			expected:    "SELECT * FROM (SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' LIMIT 10) LIMIT 5",
			shouldBlock: false,
		},
		{
			name:        "subquery with aggregation not selecting workspace_id",
			query:       "SELECT date, verifications FROM (SELECT time as date, SUM(count) as verifications FROM default.key_verifications_raw_v2 WHERE time >= now() - INTERVAL 60 DAY GROUP BY date) ORDER BY date",
			expected:    "SELECT date, verifications FROM (SELECT time AS date, SUM(count) AS verifications FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND (time >= now() - INTERVAL 60 DAY) GROUP BY date LIMIT 10) ORDER BY date LIMIT 10",
			shouldBlock: false,
		},
		{
			name:        "subquery in WHERE with system table",
			query:       "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id IN (SELECT key_id FROM system.tables)",
			shouldBlock: true,
		},
		{
			name:        "subquery with unauthorized table",
			query:       "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id IN (SELECT id FROM default.secrets)",
			shouldBlock: true,
		},
		{
			name:        "nested subquery with system table",
			query:       "SELECT * FROM (SELECT * FROM (SELECT * FROM system.tables) t1) t2",
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)

			if tt.shouldBlock {
				require.Error(t, err, "Subquery with unauthorized/system tables should be blocked")
				require.Contains(t, err.Error(), "not allowed", "Error should indicate table is not allowed")
			} else {
				require.NoError(t, err, "Subquery with allowed tables should work")
				require.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParser_CTEWithTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name        string
		query       string
		shouldBlock bool
	}{
		{
			name:        "CTE with system table",
			query:       "WITH t AS (SELECT * FROM system.tables) SELECT * FROM t",
			shouldBlock: true,
		},
		{
			name:        "CTE with unauthorized table",
			query:       "WITH secrets AS (SELECT * FROM default.secrets) SELECT * FROM secrets",
			shouldBlock: true,
		},
		{
			name:        "Multiple CTEs with one unauthorized",
			query:       "WITH t1 AS (SELECT * FROM default.key_verifications_raw_v2), t2 AS (SELECT * FROM system.tables) SELECT * FROM t1 JOIN t2 ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "Nested CTEs with system table",
			query:       "WITH t1 AS (SELECT * FROM default.key_verifications_raw_v2), t2 AS (SELECT * FROM t1), t3 AS (SELECT * FROM system.tables) SELECT * FROM t2",
			shouldBlock: true,
		},
		{
			name:        "CTE with allowed table should work",
			query:       "WITH t AS (SELECT * FROM default.key_verifications_raw_v2) SELECT * FROM t",
			shouldBlock: false,
		},
		{
			name:        "Multiple CTEs with allowed tables",
			query:       "WITH t1 AS (SELECT * FROM default.key_verifications_raw_v2), t2 AS (SELECT * FROM t1) SELECT * FROM t2",
			shouldBlock: false,
		},
		{
			name:        "CTE JOIN with allowed table",
			query:       "WITH t AS (SELECT * FROM default.key_verifications_raw_v2) SELECT * FROM t JOIN default.key_verifications_raw_v2 v ON t.key_id = v.key_id",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)

			if tt.shouldBlock {
				require.Error(t, err, "CTE with unauthorized/system tables should be blocked")
				require.Contains(t, err.Error(), "not allowed", "Error should indicate table is not allowed")
			} else {
				require.NoError(t, err, "CTE with allowed tables should work")
			}
		})
	}
}

func TestParser_ScalarSubqueryWithTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name        string
		query       string
		shouldBlock bool
	}{
		{
			name:        "scalar subquery with system table",
			query:       "SELECT key_id, (SELECT COUNT(*) FROM system.tables) AS cnt FROM default.key_verifications_raw_v2",
			shouldBlock: true,
		},
		{
			name:        "scalar subquery with unauthorized table",
			query:       "SELECT key_id, (SELECT secret FROM default.secrets LIMIT 1) AS s FROM default.key_verifications_raw_v2",
			shouldBlock: true,
		},
		{
			name:        "scalar subquery with allowed table",
			query:       "SELECT key_id, (SELECT COUNT(*) FROM default.key_verifications_raw_v2) AS cnt FROM default.key_verifications_raw_v2",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)

			if tt.shouldBlock {
				require.Error(t, err, "Scalar subquery with unauthorized/system tables should be blocked")
				require.Contains(t, err.Error(), "not allowed", "Error should indicate table is not allowed")
			} else {
				require.NoError(t, err, "Scalar subquery with allowed tables should work")
			}
		})
	}
}

func TestParser_TableFunctions(t *testing.T) {
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
			name:  "file() table function",
			query: "SELECT * FROM file('/etc/passwd', 'CSV')",
		},
		{
			name:  "url() table function",
			query: "SELECT * FROM url('http://evil.com/data', 'JSONEachRow')",
		},
		{
			name:  "remote() table function to another server",
			query: "SELECT * FROM remote('other-server:9000', 'db', 'table')",
		},
		{
			name:  "s3() table function",
			query: "SELECT * FROM s3('https://bucket.s3.amazonaws.com/data.csv')",
		},
		{
			name:  "hdfs() table function",
			query: "SELECT * FROM hdfs('hdfs://namenode:port/path')",
		},
		{
			name:  "mysql() table function",
			query: "SELECT * FROM mysql('mysql://user:pass@host/db', 'table', 'select * from sensitive')",
		},
		{
			name:  "postgresql() table function",
			query: "SELECT * FROM postgresql('postgres://host/db', 'table')",
		},
		{
			name:  "executable() table function",
			query: "SELECT * FROM executable('/bin/bash', 'CSV', 'arg1')",
		},
		{
			name:  "azureblobstorage() table function",
			query: "SELECT * FROM azureblobstorage('https://account.blob.core.windows.net/container/file')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			require.Error(t, err, "Table function should be blocked")
			require.Contains(t, err.Error(), "not allowed", "Error should indicate function is not allowed")
		})
	}
}
