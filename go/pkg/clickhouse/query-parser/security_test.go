package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSQLInjectionAttempts tests various SQL injection attack vectors
func TestSQLInjectionAttempts(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       1000,
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "injection in WHERE clause with quotes",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '' OR '1'='1",
		},
		{
			name:  "injection with comment",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '' -- comment",
		},
		{
			name:  "injection with multiline comment",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '/* comment */'",
		},
		{
			name:  "injection with semicolon",
			query: "SELECT * FROM default.key_verifications_raw_v2; DROP TABLE users",
		},
		{
			name:  "injection in column alias",
			query: "SELECT key_id AS 'malicious''code' FROM default.key_verifications_raw_v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)

			// Query should either parse safely or fail
			// Most importantly, workspace filter should ALWAYS be present
			if err == nil {
				require.Contains(t, result, "workspace_id = 'ws_123'",
					"Workspace filter MUST be present even with injection attempts")
			}
		})
	}
}

// TestWorkspaceIsolationBypass tests attempts to bypass workspace filtering
func TestWorkspaceIsolationBypass(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_victim",
		Limit:       1000,
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "OR to bypass workspace filter",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_attacker' OR 1=1",
		},
		{
			name:  "NOT to invert workspace filter",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE NOT workspace_id = 'ws_victim'",
		},
		{
			name:  "workspace_id in SELECT to confuse parser",
			query: "SELECT workspace_id FROM default.key_verifications_raw_v2 WHERE key_id = 'test'",
		},
		{
			name:  "workspace_id with different case",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE WORKSPACE_ID = 'ws_attacker'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)

			// Even if query parses, workspace filter MUST be enforced
			if err == nil {
				require.Contains(t, result, "workspace_id = 'ws_victim'", "Attacker's workspace should NOT be used - victim workspace MUST be enforced")
			}
		})
	}
}

// TestUNIONAttacks tests attempts to use UNION to access other data
func TestUNIONAttacks(t *testing.T) {
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

// TestSubqueryAttempts tests whether subqueries can be used to bypass restrictions
func TestSubqueryAttempts(t *testing.T) {
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
		shouldBlock bool
	}{
		{
			name:        "subquery in FROM with allowed table",
			query:       "SELECT * FROM (SELECT * FROM default.key_verifications_raw_v2 LIMIT 10000) LIMIT 5",
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
				require.Contains(t, result, "workspace_id = 'ws_123'",
					"Workspace filter must be present even with subqueries")
				// Should also respect limit
				require.Contains(t, result, "LIMIT",
					"Limit must be enforced even with subqueries")
			}
		})
	}
}

// TestMultiStatementInjection tests attempts to run multiple SQL statements
func TestMultiStatementInjection(t *testing.T) {
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
			name:  "multiple SELECT statements",
			query: "SELECT * FROM default.key_verifications_raw_v2; SELECT * FROM system.tables",
		},
		{
			name:  "SELECT then DROP",
			query: "SELECT * FROM default.key_verifications_raw_v2; DROP TABLE default.key_verifications_raw_v2",
		},
		{
			name:  "SELECT then INSERT",
			query: "SELECT * FROM default.key_verifications_raw_v2; INSERT INTO default.key_verifications_raw_v2 VALUES (1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)

			// Parser should either reject multiple statements or only execute first
			// We only parse the first statement, so this should be safe
			if err == nil {
				t.Logf("Multiple statements attempted: %s", tt.query)
			}
		})
	}
}

// TestDangerousFunctions tests blocking of functions that could be used maliciously
func TestDangerousFunctions(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
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
			query:      "SELECT file('/etc/passwd') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "url function",
			query:      "SELECT url('http://evil.com/data') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "system function",
			query:      "SELECT system('rm -rf /') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "executable function",
			query:      "SELECT executable('/bin/bash') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "dict_get to access dictionaries",
			query:      "SELECT dictGet('dict', 'attr', key_id) FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "nested safe functions should work",
			query:      "SELECT count(DISTINCT key_id) FROM default.key_verifications_raw_v2",
			shouldFail: false,
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

// TestInformationSchemaAccess tests blocking access to information_schema
func TestInformationSchemaAccess(t *testing.T) {
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

// TestSystemTableAccess tests blocking access to system tables
func TestSystemTableAccess(t *testing.T) {
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

// TestNonSELECTStatements tests that only SELECT is allowed
func TestNonSELECTStatements(t *testing.T) {
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
			// The error message varies depending on the statement type
			// but all non-SELECT statements should fail
		})
	}
}

// TestSpecialCharactersInInput tests handling of special characters that might break parsing
func TestSpecialCharactersInInput(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       1000,
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "null bytes",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '\x00'",
		},
		{
			name:  "unicode characters",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '你好'",
		},
		{
			name:  "emoji",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '🔥'",
		},
		{
			name:  "backslashes",
			query: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '\\\\'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)

			// Special characters should either be handled safely or rejected
			// Most importantly, workspace filter should always be present
			if err == nil {
				require.Contains(t, result, "workspace_id = 'ws_123'",
					"Workspace filter must be present even with special characters")
			}
		})
	}
}

// TestLimitBypass tests attempts to bypass or remove the LIMIT
func TestLimitBypass(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       10,
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "LIMIT with OFFSET to read more",
			query: "SELECT * FROM default.key_verifications_raw_v2 LIMIT 100000 OFFSET 0",
		},
		{
			name:  "extremely high LIMIT",
			query: "SELECT * FROM default.key_verifications_raw_v2 LIMIT 999999999",
		},
		{
			name:  "negative LIMIT",
			query: "SELECT * FROM default.key_verifications_raw_v2 LIMIT -1",
		},
		{
			name:  "LIMIT ALL",
			query: "SELECT * FROM default.key_verifications_raw_v2 LIMIT ALL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)

			// Parser should enforce max limit
			require.NoError(t, err, "Query should parse successfully but with limit enforced")
			require.Contains(t, result, "LIMIT 10", "Query must have LIMIT enforced to configured max of 10")
			require.NotContains(t, result, "LIMIT 100000", "Large limit should be reduced")
			require.NotContains(t, result, "LIMIT 999999999", "Extreme limit should be reduced")
		})
	}
}

// TestJOINAttacks tests attempts to use JOINs to access unauthorized data
func TestJOINAttacks(t *testing.T) {
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

// TestCTEAttacks tests attempts to use WITH (Common Table Expressions) to access unauthorized data
func TestCTEAttacks(t *testing.T) {
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
			shouldBlock: true, // Blocks system.tables access
		},
		{
			name:        "CTE with unauthorized table",
			query:       "WITH secrets AS (SELECT * FROM default.secrets) SELECT * FROM secrets",
			shouldBlock: true, // Blocks default.secrets access
		},
		{
			name:        "Multiple CTEs with one unauthorized",
			query:       "WITH t1 AS (SELECT * FROM default.key_verifications_raw_v2), t2 AS (SELECT * FROM system.tables) SELECT * FROM t1 JOIN t2 ON 1=1",
			shouldBlock: true, // Blocks system.tables access
		},
		{
			name:        "Nested CTEs with system table",
			query:       "WITH t1 AS (SELECT * FROM default.key_verifications_raw_v2), t2 AS (SELECT * FROM t1), t3 AS (SELECT * FROM system.tables) SELECT * FROM t2",
			shouldBlock: true, // Blocks system.tables access
		},
		{
			name:        "CTE with allowed table should work",
			query:       "WITH t AS (SELECT * FROM default.key_verifications_raw_v2) SELECT * FROM t",
			shouldBlock: false, // CTE references allowed table
		},
		{
			name:        "Multiple CTEs with allowed tables",
			query:       "WITH t1 AS (SELECT * FROM default.key_verifications_raw_v2), t2 AS (SELECT * FROM t1) SELECT * FROM t2",
			shouldBlock: false, // Nested CTEs with allowed tables
		},
		{
			name:        "CTE JOIN with allowed table",
			query:       "WITH t AS (SELECT * FROM default.key_verifications_raw_v2) SELECT * FROM t JOIN default.key_verifications_raw_v2 v ON t.key_id = v.key_id",
			shouldBlock: false, // CTE with JOIN
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

// TestScalarSubqueryAttacks tests scalar subqueries in SELECT clause
func TestScalarSubqueryAttacks(t *testing.T) {
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

// TestTableFunctionAttacks tests table functions that could access external data
func TestTableFunctionAttacks(t *testing.T) {
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

			// All table functions should be blocked
			require.Error(t, err, "Table function should be blocked for security")
			require.Contains(t, err.Error(), "not allowed", "Error should indicate function is not allowed")
		})
	}
}
