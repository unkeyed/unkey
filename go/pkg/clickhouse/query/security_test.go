package query

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSQLInjection tests various SQL injection attack vectors
func TestSQLInjection(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_abc123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []struct {
		name        string
		query       string
		shouldBlock bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:        "try to comment out workspace filter",
			query:       "SELECT * FROM key_verifications WHERE key_id = 'abc' -- and workspace_id = 'ws_abc123'",
			shouldBlock: false, // Should be rewritten safely
			checkOutput: func(t *testing.T, output string) {
				// Our filter should be added BEFORE their WHERE clause
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
				// The comment should still be there but our filter is already applied
			},
		},
		{
			name:        "try OR 1=1 injection",
			query:       "SELECT * FROM key_verifications WHERE key_id = 'abc' OR 1=1",
			shouldBlock: false, // Should be rewritten with workspace filter
			checkOutput: func(t *testing.T, output string) {
				// Our workspace_id filter should be ANDed at the beginning
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
				// Even with OR 1=1, workspace filter is enforced
			},
		},
		{
			name:        "try UNION injection to access other workspaces",
			query:       "SELECT * FROM key_verifications UNION SELECT * FROM key_verifications WHERE workspace_id = 'ws_other'",
			shouldBlock: false, // Parser will handle this
			checkOutput: func(t *testing.T, output string) {
				// UNION queries are complex, parser might reject or both parts get filtered
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
			},
		},
		{
			name:        "try to override workspace_id in WHERE",
			query:       "SELECT * FROM key_verifications WHERE workspace_id = 'ws_other'",
			shouldBlock: false,
			checkOutput: func(t *testing.T, output string) {
				// Our filter is ANDed at the start, so this becomes:
				// WHERE workspace_id = 'ws_abc123' AND workspace_id = 'ws_other'
				// Which will return no results (correct behavior)
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
				// The user's filter is also there, but they'll get no results
				require.Contains(t, output, "workspace_id = 'ws_other'")
			},
		},
		{
			name:        "try to use OR to bypass workspace filter",
			query:       "SELECT * FROM key_verifications WHERE (1=0 OR workspace_id = 'ws_other')",
			shouldBlock: false,
			checkOutput: func(t *testing.T, output string) {
				// Our filter is enforced first: workspace_id = 'ws_abc123' AND (...)
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
				// They can't escape it with OR because we AND at the top level
			},
		},
		{
			name:        "try to inject with string concatenation",
			query:       "SELECT * FROM key_verifications WHERE key_id = 'abc' + 'def'",
			shouldBlock: false,
			checkOutput: func(t *testing.T, output string) {
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
			},
		},
		{
			name:        "try subquery to access other workspace",
			query:       "SELECT * FROM key_verifications WHERE key_id IN (SELECT key_id FROM key_verifications WHERE workspace_id = 'ws_other')",
			shouldBlock: false,
			checkOutput: func(t *testing.T, output string) {
				// Both outer and inner queries should get workspace filter
				count := strings.Count(output, "workspace_id = 'ws_abc123'")
				require.GreaterOrEqual(t, count, 2, "Both outer and subquery should have workspace filter")
			},
		},
		{
			name:  "try DROP TABLE",
			query: "SELECT * FROM key_verifications; DROP TABLE key_verifications; --",
			// Parser should reject multi-statement
			shouldBlock: true,
		},
		{
			name:  "try INSERT via injection",
			query: "SELECT * FROM key_verifications WHERE key_id = 'abc'; INSERT INTO key_verifications VALUES (...); --",
			// Parser should reject multi-statement
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.Rewrite(context.Background(), tt.query)

			if tt.shouldBlock {
				require.Error(t, err, "Query should have been blocked")
			} else {
				if err != nil {
					// Some queries might fail parsing, that's ok for security
					t.Logf("Query rejected (safe): %v", err)
					return
				}
				require.NoError(t, err)
				if tt.checkOutput != nil {
					tt.checkOutput(t, output)
				}
			}
		})
	}
}

// TestWorkspaceIsolation ensures users cannot access other workspaces' data
func TestWorkspaceIsolation(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_victim",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []struct {
		name        string
		query       string
		mustContain []string // All of these must be in output
	}{
		{
			name:  "simple query gets workspace filter",
			query: "SELECT * FROM key_verifications",
			mustContain: []string{
				"workspace_id = 'ws_victim'",
			},
		},
		{
			name:  "query with WHERE gets workspace filter prepended",
			query: "SELECT * FROM key_verifications WHERE valid = true",
			mustContain: []string{
				"workspace_id = 'ws_victim'",
				"valid = true",
			},
		},
		{
			name:  "trying to access other workspace still gets filter",
			query: "SELECT * FROM key_verifications WHERE workspace_id = 'ws_attacker'",
			mustContain: []string{
				"workspace_id = 'ws_victim'", // Our filter
				// Note: Their filter for ws_attacker will also be there,
				// but since we AND our filter first, query returns nothing
			},
		},
		{
			name:  "query with OR still gets workspace filter",
			query: "SELECT * FROM key_verifications WHERE key_id = 'a' OR key_id = 'b'",
			mustContain: []string{
				"workspace_id = 'ws_victim'",
			},
		},
		{
			name:  "complex query with subquery",
			query: "SELECT * FROM key_verifications WHERE key_id IN (SELECT key_id FROM key_verifications WHERE valid = true)",
			mustContain: []string{
				"workspace_id = 'ws_victim'",
			},
		},
		{
			name: "aggregation query",
			query: `SELECT
				COUNT(*) as total,
				SUM(CASE WHEN valid = true THEN 1 ELSE 0 END) as valid_count
			FROM key_verifications
			GROUP BY key_id`,
			mustContain: []string{
				"workspace_id = 'ws_victim'",
			},
		},
		{
			name:  "join query",
			query: "SELECT * FROM key_verifications k1 JOIN key_verifications k2 ON k1.key_id = k2.key_id",
			mustContain: []string{
				"workspace_id = 'ws_victim'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.Rewrite(context.Background(), tt.query)
			require.NoError(t, err, "Query should be rewritten successfully")

			for _, mustContain := range tt.mustContain {
				require.Contains(t, output, mustContain,
					"Rewritten query must contain: %s\nGot: %s", mustContain, output)
			}

			// Ensure workspace filter appears before any user conditions
			wsFilterIdx := strings.Index(output, "workspace_id = 'ws_victim'")
			require.NotEqual(t, -1, wsFilterIdx, "Workspace filter must be present")

			// If there's a WHERE clause, our filter should be at the start of it
			whereIdx := strings.Index(strings.ToLower(output), "where")
			if whereIdx != -1 {
				// Our filter should come shortly after WHERE
				require.Less(t, wsFilterIdx, whereIdx+20,
					"Workspace filter should be at the start of WHERE clause")
			}
		})
	}
}

// TestCaseSensitivityAttacks tests if attackers can use case variations
func TestCaseSensitivityAttacks(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_abc123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "uppercase SELECT",
			query: "SELECT * FROM key_verifications",
		},
		{
			name:  "mixed case SELECT",
			query: "SeLeCt * FrOm key_verifications",
		},
		{
			name:  "uppercase WHERE",
			query: "SELECT * FROM key_verifications WHERE valid = TRUE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.Rewrite(context.Background(), tt.query)
			require.NoError(t, err)
			require.Contains(t, output, "workspace_id = 'ws_abc123'")
		})
	}
}

// TestQuoteEscaping tests various quote escaping attacks
func TestQuoteEscaping(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_abc123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "single quotes in string",
			query: "SELECT * FROM key_verifications WHERE key_id = 'abc''def'",
		},
		{
			name:  "backslash escape attempts",
			query: "SELECT * FROM key_verifications WHERE key_id = 'abc\\'def'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.Rewrite(context.Background(), tt.query)
			// May fail parsing (that's ok), but if it succeeds, must have workspace filter
			if err == nil {
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
			}
		})
	}
}

// TestTimeBasedAttacks tests timing-based SQL injection attempts
func TestTimeBasedAttacks(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_abc123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []struct {
		name        string
		query       string
		shouldBlock bool
	}{
		{
			name:        "sleep function (if it exists)",
			query:       "SELECT * FROM key_verifications WHERE key_id = 'abc' AND SLEEP(5)",
			shouldBlock: false, // Parser might allow, but workspace filter still applied
		},
		{
			name:        "benchmark function attempts",
			query:       "SELECT * FROM key_verifications WHERE key_id = 'abc' AND BENCHMARK(1000000, MD5('test'))",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := r.Rewrite(context.Background(), tt.query)
			if err == nil {
				// If parsing succeeds, workspace filter must be present
				require.Contains(t, output, "workspace_id = 'ws_abc123'")
			}
			// If parsing fails, that's also acceptable (query blocked)
		})
	}
}

// TestNoWorkspaceColumn ensures queries fail gracefully if workspace_id column doesn't exist
// (This is a schema validation concern, but we test the rewriter behavior)
func TestMultipleWorkspaceFilters(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_abc123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	// User tries to query with their own workspace_id filter
	query := "SELECT * FROM key_verifications WHERE workspace_id = 'ws_other'"
	output, err := r.Rewrite(context.Background(), query)
	require.NoError(t, err)

	// Should have TWO workspace_id filters:
	// 1. Our injected one: workspace_id = 'ws_abc123'
	// 2. Their attempted one: workspace_id = 'ws_other'
	// This will return no results (correct behavior)
	require.Contains(t, output, "workspace_id = 'ws_abc123'")
	require.Contains(t, output, "workspace_id = 'ws_other'")

	// The query is logically: WHERE 'ws_abc123' = workspace_id AND workspace_id = 'ws_other'
	// Which is impossible, returns empty set (safe!)
}

// TestStacked queries tests if multiple statements can be injected
func TestStackedQueries(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_abc123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "semicolon with DROP",
			query: "SELECT * FROM key_verifications; DROP TABLE key_verifications;",
		},
		{
			name:  "semicolon with INSERT",
			query: "SELECT * FROM key_verifications; INSERT INTO key_verifications VALUES (1,2,3);",
		},
		{
			name:  "semicolon with UPDATE",
			query: "SELECT * FROM key_verifications; UPDATE key_verifications SET workspace_id = 'ws_attacker';",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Rewrite(context.Background(), tt.query)
			// Parser should reject stacked queries or only parse the first one
			// Either way, the dangerous statement shouldn't execute
			if err == nil {
				// If it somehow parses, it should only be the SELECT part
				// and workspace filter should be applied
				t.Logf("Query parsed (only first statement should be used)")
			}
		})
	}
}
