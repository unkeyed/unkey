package queryparser_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	chquery "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
)

func TestParser_WorkspaceFilter(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"keys_v1": "default.keys_v2",
		},
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM keys_v1")
	require.NoError(t, err)

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE keys_v2.workspace_id = 'ws_123'", output)
}

func TestParser_WorkspaceFilterWithExistingWhere(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_456",
		TableAliases: map[string]string{
			"keys_v1": "default.keys_v2",
		},
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM keys_v1 WHERE active = 1")
	require.NoError(t, err)

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE keys_v2.workspace_id = 'ws_456' AND (active = 1)", output)
}

// TestParser_QualifiesFiltersForEveryPhysicalSource guarantees aliases,
// subqueries, CTEs, UNIONs, and joins cannot leave a physical source unscoped.
func TestParser_QualifiesFiltersForEveryPhysicalSource(t *testing.T) {
	config := chquery.Config{
		WorkspaceID:     "ws_safe",
		TableAliases:    map[string]string{"events_v1": "default.events", "other_v1": "default.other", "ratelimits_v1": "default.ratelimits_raw_v2"},
		AllowedTables:   []string{"default.events", "default.other", "default.ratelimits_raw_v2"},
		SecurityFilters: []chquery.SecurityFilter{{Column: "namespace_id", AllowedValues: []string{"ns_safe"}}},
	}
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"expression alias collision", "SELECT 1 AS workspace_id FROM events_v1 e", []string{"e.workspace_id = 'ws_safe'", "e.namespace_id IN ('ns_safe')"}},
		{"scalar WITH alias collision", "WITH (SELECT 1) AS namespace_id SELECT * FROM events_v1 e", []string{"e.workspace_id = 'ws_safe'", "e.namespace_id IN ('ns_safe')"}},
		{"table plus scalar subquery", "SELECT (SELECT 1) FROM events_v1 e CROSS JOIN (SELECT 1) x WHERE 1=0 OR 1=1", []string{"e.workspace_id = 'ws_safe'", "e.namespace_id IN ('ns_safe')"}},
		{"two physical tables", "SELECT * FROM events_v1 e CROSS JOIN other_v1 o", []string{"e.workspace_id = 'ws_safe'", "o.workspace_id = 'ws_safe'", "e.namespace_id IN ('ns_safe')", "o.namespace_id IN ('ns_safe')"}},
		{"table joined to filtered subquery", "SELECT * FROM events_v1 e CROSS JOIN (SELECT * FROM other_v1) o", []string{"e.workspace_id = 'ws_safe'", "other.workspace_id = 'ws_safe'", "e.namespace_id IN ('ns_safe')", "other.namespace_id IN ('ns_safe')"}},
		{"CTE filters inner physical source only", "WITH c AS (SELECT * FROM events_v1) SELECT * FROM c", []string{"events.workspace_id = 'ws_safe'", "events.namespace_id IN ('ns_safe')"}},
		{"unused CTE collides with rewritten physical table", "WITH ratelimits_raw_v2 AS (SELECT * FROM other_v1) SELECT * FROM ratelimits_v1 WHERE 1=0 OR 1=1", []string{"ratelimits_raw_v2.workspace_id = 'ws_safe'", "ratelimits_raw_v2.namespace_id IN ('ns_safe')"}},
		{"UNION filters both branches", "SELECT * FROM events_v1 e UNION ALL SELECT * FROM other_v1 o", []string{"e.workspace_id = 'ws_safe'", "o.workspace_id = 'ws_safe'", "e.namespace_id IN ('ns_safe')", "o.namespace_id IN ('ns_safe')"}},
		{"EXCEPT filters both branches", "SELECT * FROM events_v1 e EXCEPT SELECT * FROM other_v1 o", []string{"e.workspace_id = 'ws_safe'", "o.workspace_id = 'ws_safe'", "e.namespace_id IN ('ns_safe')", "o.namespace_id IN ('ns_safe')"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output, err := chquery.NewParser(config).Parse(context.Background(), test.query)
			require.NoError(t, err)
			for _, want := range test.want {
				require.Contains(t, output, want)
			}
			if test.name == "CTE filters inner physical source only" {
				require.NotContains(t, output, "c.workspace_id")
				require.NotContains(t, output, "c.namespace_id")
			}
		})
	}
}

// TestParser_PreservesQuotedAliasOnInjectedFilters guarantees generated
// predicates remain valid when callers use aliases requiring identifier quotes.
func TestParser_PreservesQuotedAliasOnInjectedFilters(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		WorkspaceID:     "ws_safe",
		TableAliases:    map[string]string{"events_v1": "default.events"},
		AllowedTables:   []string{"default.events"},
		SecurityFilters: []chquery.SecurityFilter{{Column: "namespace_id", AllowedValues: []string{"ns_safe"}}},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM events_v1 AS `r-l`")
	require.NoError(t, err)
	require.Contains(t, output, "`r-l`.workspace_id = 'ws_safe'")
	require.Contains(t, output, "`r-l`.namespace_id IN ('ns_safe')")
}

func TestSecurityFilterInjection(t *testing.T) {
	t.Run("no filter when SecurityFilters is empty", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID:     "ws_test",
			SecurityFilters: nil, // No restriction
			Limit:           100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_test' LIMIT 100", result)
	})

	t.Run("fails closed when a filter has no allowed values", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_id",
					AllowedValues: []string{}, // Scoped principal that may see nothing
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Empty allowed values must render as a constant-false predicate, not be
		// dropped, so the query returns zero rows instead of leaking the workspace.
		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_test' AND (0) LIMIT 100", result)
	})

	t.Run("injects single key_space_id filter", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_test' AND (key_verifications_raw_v2.key_space_id IN ('ks_123')) LIMIT 100", result)
	})

	t.Run("injects multiple key_space_id filter", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123", "ks_456", "ks_789"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_test' AND (key_verifications_raw_v2.key_space_id IN ('ks_123', 'ks_456', 'ks_789')) LIMIT 100", result)
	})

	t.Run("combines with existing WHERE clause", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications WHERE outcome = 'VALID'"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_test' AND (key_verifications_raw_v2.key_space_id IN ('ks_123') AND (outcome = 'VALID')) LIMIT 100", result)
	})

	t.Run("restricts access even when user queries different key_space_id", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123"}, // User only has access to ks_123
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		// User tries to query ks_999 which they don't have access to
		query := "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_999'"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Both filters are present, creating impossible AND condition
		// Injected: key_space_id IN ('ks_123') - only ks_123
		// User's: key_space_id = 'ks_999'
		// Result: no rows (ks_123 AND ks_999 = impossible)
		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_test' AND (key_verifications_raw_v2.key_space_id IN ('ks_123') AND (key_space_id = 'ks_999')) LIMIT 100", result)
	})

	t.Run("OR cannot bypass security filter", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_id",
					AllowedValues: []string{"key_123"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		// Attacker appends `OR 1=1` to try to widen the result set past their key.
		// The injected key_id filter must stay ANDed against the parenthesized
		// user WHERE, so the OR cannot reassociate above the security filter.
		query := "SELECT COUNT(*) FROM key_verifications WHERE key_id = 'key_999' OR 1=1"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_test' AND (key_verifications_raw_v2.key_id IN ('key_123') AND (key_id = 'key_999' OR 1 = 1)) LIMIT 100", result)
	})

	t.Run("supports multiple security filters simultaneously", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123", "ks_456"},
				},
				{
					Column:        "namespace_id",
					AllowedValues: []string{"nsid_111", "nsid_222"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"ratelimits": "default.ratelimits_v2",
			},
			AllowedTables: []string{
				"default.ratelimits_v2",
			},
		})

		query := "SELECT COUNT(*) FROM ratelimits"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.ratelimits_v2 WHERE ratelimits_v2.workspace_id = 'ws_test' AND (ratelimits_v2.namespace_id IN ('nsid_111', 'nsid_222') AND (ratelimits_v2.key_space_id IN ('ks_123', 'ks_456'))) LIMIT 100", result)
	})
}

func TestParser_WorkspaceFilterInjection(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_victim",
		Limit:       1000,
		TableAliases: map[string]string{
			"key_verifications_v1": "default.key_verifications_raw_v2",
		},
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "OR cannot bypass workspace filter",
			query:    "SELECT * FROM key_verifications_v1 WHERE workspace_id = 'ws_attacker' OR 1=1",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_victim' AND (workspace_id = 'ws_attacker' OR 1 = 1) LIMIT 1000",
		},
		{
			name:     "NOT cannot invert workspace filter",
			query:    "SELECT * FROM key_verifications_v1 WHERE NOT workspace_id = 'ws_victim'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_victim' AND (NOT workspace_id = 'ws_victim') LIMIT 1000",
		},
		{
			name:     "workspace_id in SELECT to confuse parser",
			query:    "SELECT workspace_id FROM key_verifications_v1 WHERE key_id = 'test'",
			expected: "SELECT workspace_id FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_victim' AND (key_id = 'test') LIMIT 1000",
		},
		{
			name:     "workspace_id with different case",
			query:    "SELECT * FROM key_verifications_v1 WHERE WORKSPACE_ID = 'ws_attacker'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_victim' AND (WORKSPACE_ID = 'ws_attacker') LIMIT 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestParser_SQLInjectionWithFilters guarantees SQL-looking filter values stay
// inside the rewritten predicate and additional statements are rejected.
func TestParser_SQLInjectionWithFilters(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
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
		name     string
		query    string
		expected string
	}{
		{
			name:     "injection in WHERE clause with quotes",
			query:    "SELECT * FROM key_verifications_v1 WHERE key_id = '' OR '1'='1'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (key_id = '' OR '1' = '1') LIMIT 1000",
		},
		{
			name:     "injection with comment",
			query:    "SELECT * FROM key_verifications_v1 WHERE key_id = '' -- comment",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (key_id = '') LIMIT 1000",
		},
		{
			name:     "injection with multiline comment",
			query:    "SELECT * FROM key_verifications_v1 WHERE key_id = '/* comment */'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (key_id = '/* comment */') LIMIT 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}

	_, err := p.Parse(context.Background(), "SELECT * FROM key_verifications_v1; DROP TABLE users")
	require.Error(t, err)
}

func TestParser_SpecialCharactersInFilters(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
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
		name     string
		query    string
		expected string
	}{
		{
			name:     "null bytes",
			query:    "SELECT * FROM key_verifications_v1 WHERE key_id = '\x00'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (key_id = '\x00') LIMIT 1000",
		},
		{
			name:     "unicode characters",
			query:    "SELECT * FROM key_verifications_v1 WHERE key_id = '你好'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (key_id = '你好') LIMIT 1000",
		},
		{
			name:     "emoji",
			query:    "SELECT * FROM key_verifications_v1 WHERE key_id = '🔥'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (key_id = '🔥') LIMIT 1000",
		},
		{
			name:     "backslashes",
			query:    "SELECT * FROM key_verifications_v1 WHERE key_id = '\\\\'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = 'ws_123' AND (key_id = '\\\\') LIMIT 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := p.Parse(context.Background(), tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}
