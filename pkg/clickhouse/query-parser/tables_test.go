package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE keys_v2.workspace_id = 'ws_123'", output)
}

// TestParser_RejectsCTEsThatShadowPublicAliases guarantees a CTE reference
// cannot be silently replaced with the physical table behind a public alias.
func TestParser_RejectsCTEsThatShadowPublicAliases(t *testing.T) {
	tests := []struct {
		name          string
		publicAlias   string
		physicalTable string
		query         string
	}{
		{
			name:          "verification alias",
			publicAlias:   "key_verifications_v1",
			physicalTable: "default.key_verifications_raw_v2",
			query:         "WITH key_verifications_v1 AS (SELECT 1 AS total) SELECT count() FROM key_verifications_v1",
		},
		{
			name:          "rate limit alias",
			publicAlias:   "ratelimits_v1",
			physicalTable: "default.ratelimits_raw_v2",
			query:         "WITH ratelimits_v1 AS (SELECT 'rlns_123' AS namespace_id) SELECT count() FROM ratelimits_v1",
		},
		{
			name:          "nested verification alias",
			publicAlias:   "key_verifications_v1",
			physicalTable: "default.key_verifications_raw_v2",
			query:         "SELECT * FROM (WITH key_verifications_v1 AS (SELECT 1 AS total) SELECT total FROM key_verifications_v1)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := NewParser(Config{
				WorkspaceID: "ws_123",
				TableAliases: map[string]string{
					test.publicAlias: test.physicalTable,
				},
				AllowedTables: []string{test.physicalTable},
			})

			_, err := parser.Parse(context.Background(), test.query)
			require.Error(t, err)
		})
	}
}

// TestParser_RequiresPublicTableAliases guarantees every physical source,
// including nested and UNION sources, is introduced through a configured alias.
func TestParser_RequiresPublicTableAliases(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"ratelimits_v1":            "default.ratelimits_raw_v2",
			"ratelimits_per_minute_v1": "default.ratelimits_per_minute_v2",
			"ratelimits_per_hour_v1":   "default.ratelimits_per_hour_v2",
			"ratelimits_per_day_v1":    "default.ratelimits_per_day_v2",
			"ratelimits_per_month_v1":  "default.ratelimits_per_month_v2",
		},
		AllowedTables: []string{"default.ratelimits_raw_v2", "default.ratelimits_per_minute_v2", "default.ratelimits_per_hour_v2", "default.ratelimits_per_day_v2", "default.ratelimits_per_month_v2"},
	})

	for _, alias := range []string{"ratelimits_v1", "ratelimits_per_minute_v1", "ratelimits_per_hour_v1", "ratelimits_per_day_v1", "ratelimits_per_month_v1"} {
		_, err := p.Parse(context.Background(), "SELECT * FROM "+alias)
		require.NoError(t, err, alias)
	}
	for _, query := range []string{
		"SELECT * FROM ratelimits_v1 r JOIN ratelimits_per_hour_v1 h USING namespace_id",
		"SELECT * FROM ratelimits_v1 EXCEPT SELECT * FROM ratelimits_v1",
	} {
		_, err := p.Parse(context.Background(), query)
		require.NoError(t, err, query)
	}
	var err error
	queries := []string{
		"SELECT * FROM default.ratelimits_raw_v2",
		"SELECT * FROM ratelimits_v1 JOIN default.ratelimits_raw_v2 physical USING namespace_id",
		"SELECT * FROM (SELECT * FROM default.ratelimits_raw_v2)",
		"WITH hidden AS (SELECT * FROM default.ratelimits_raw_v2) SELECT * FROM hidden",
		"SELECT * FROM ratelimits_v1 UNION ALL SELECT * FROM default.ratelimits_raw_v2",
		"SELECT * FROM ratelimits_v1 EXCEPT SELECT * FROM default.ratelimits_raw_v2",
	}
	for _, query := range queries {
		_, err = p.Parse(context.Background(), query)
		require.Error(t, err, query)
	}
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
		TableAliases: map[string]string{
			"keys":  "default.keys_v2",
			"other": "default.other_table",
		},
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	// Allowed table should work
	_, err := p.Parse(context.Background(), "SELECT * FROM keys")
	require.NoError(t, err)

	// Non-allowed table should fail
	_, err = p.Parse(context.Background(), "SELECT * FROM other")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
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
		TableAliases: map[string]string{
			"key_verifications_v1": "default.key_verifications_raw_v2",
		},
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
			query:       "SELECT key_id FROM key_verifications_v1 UNION DISTINCT SELECT name FROM system.tables",
			shouldBlock: true,
		},
		{
			name:        "UNION ALL to bypass deduplication",
			query:       "SELECT key_id FROM key_verifications_v1 UNION ALL SELECT name FROM system.databases",
			shouldBlock: true,
		},
		{
			name:        "UNION DISTINCT with unauthorized table",
			query:       "SELECT key_id FROM key_verifications_v1 UNION DISTINCT SELECT id FROM default.secrets",
			shouldBlock: true,
		},
		{
			name:        "UNION ALL with another allowed table",
			query:       "SELECT key_id FROM key_verifications_v1 UNION ALL SELECT key_id FROM key_verifications_v1",
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
		TableAliases: map[string]string{
			"key_verifications_v1": "default.key_verifications_raw_v2",
		},
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
			query:       "SELECT * FROM key_verifications_v1 INNER JOIN system.tables ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "LEFT JOIN to unauthorized table",
			query:       "SELECT * FROM key_verifications_v1 LEFT JOIN default.secrets ON key_id = id",
			shouldBlock: true,
		},
		{
			name:        "RIGHT JOIN to system table",
			query:       "SELECT * FROM key_verifications_v1 RIGHT JOIN system.databases ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "CROSS JOIN to system table",
			query:       "SELECT * FROM key_verifications_v1 CROSS JOIN system.users",
			shouldBlock: true,
		},
		{
			name:        "Multiple JOINs with one unauthorized",
			query:       "SELECT * FROM key_verifications_v1 t1 JOIN key_verifications_v1 t2 ON t1.key_id = t2.key_id JOIN system.tables t3 ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "JOIN with allowed tables",
			query:       "SELECT * FROM key_verifications_v1 t1 INNER JOIN key_verifications_v1 t2 ON t1.key_id = t2.key_id",
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
		TableAliases: map[string]string{
			"key_verifications_v1": "default.key_verifications_raw_v2",
		},
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
			query:       "SELECT * FROM (SELECT * FROM key_verifications_v1 LIMIT 10000) LIMIT 5",
			expected:    "SELECT * FROM (SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' LIMIT 10) LIMIT 5",
			shouldBlock: false,
		},
		{
			name:        "subquery with aggregation not selecting workspace_id",
			query:       "SELECT date, verifications FROM (SELECT time as date, SUM(count) as verifications FROM key_verifications_v1 WHERE time >= now() - INTERVAL 60 DAY GROUP BY date) ORDER BY date",
			expected:    "SELECT date, verifications FROM (SELECT time AS date, SUM(count) AS verifications FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (time >= now() - INTERVAL 60 DAY) GROUP BY date LIMIT 10) ORDER BY date LIMIT 10",
			shouldBlock: false,
		},
		{
			name:        "subquery in WHERE with system table",
			query:       "SELECT * FROM key_verifications_v1 WHERE key_id IN (SELECT key_id FROM system.tables)",
			shouldBlock: true,
		},
		{
			name:        "subquery with unauthorized table",
			query:       "SELECT * FROM key_verifications_v1 WHERE key_id IN (SELECT id FROM default.secrets)",
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
		TableAliases: map[string]string{
			"key_verifications_v1": "default.key_verifications_raw_v2",
		},
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
			query:       "WITH t1 AS (SELECT * FROM key_verifications_v1), t2 AS (SELECT * FROM system.tables) SELECT * FROM t1 JOIN t2 ON 1=1",
			shouldBlock: true,
		},
		{
			name:        "Nested CTEs with system table",
			query:       "WITH t1 AS (SELECT * FROM key_verifications_v1), t2 AS (SELECT * FROM t1), t3 AS (SELECT * FROM system.tables) SELECT * FROM t2",
			shouldBlock: true,
		},
		{
			name:        "CTE with allowed table should work",
			query:       "WITH t AS (SELECT * FROM key_verifications_v1) SELECT * FROM t",
			shouldBlock: false,
		},
		{
			name:        "Multiple CTEs with allowed tables",
			query:       "WITH t1 AS (SELECT * FROM key_verifications_v1), t2 AS (SELECT * FROM t1) SELECT * FROM t2",
			shouldBlock: false,
		},
		{
			name:        "CTE JOIN with allowed table",
			query:       "WITH t AS (SELECT * FROM key_verifications_v1) SELECT * FROM t JOIN key_verifications_v1 v ON t.key_id = v.key_id",
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
		TableAliases: map[string]string{
			"key_verifications_v1": "default.key_verifications_raw_v2",
		},
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
			query:       "SELECT key_id, (SELECT COUNT(*) FROM system.tables) AS cnt FROM key_verifications_v1",
			shouldBlock: true,
		},
		{
			name:        "scalar subquery with unauthorized table",
			query:       "SELECT key_id, (SELECT secret FROM default.secrets LIMIT 1) AS s FROM key_verifications_v1",
			shouldBlock: true,
		},
		{
			name:        "scalar subquery with allowed table",
			query:       "SELECT key_id, (SELECT COUNT(*) FROM key_verifications_v1) AS cnt FROM key_verifications_v1",
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
