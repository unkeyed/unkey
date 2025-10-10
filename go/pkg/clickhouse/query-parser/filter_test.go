package queryparser_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	chquery "github.com/unkeyed/unkey/go/pkg/clickhouse/query-parser"
)

func TestParser_WorkspaceFilter(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2")
	require.NoError(t, err)

	require.Contains(t, output, "workspace_id = 'ws_123'")
}

func TestParser_WorkspaceFilterWithExistingWhere(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_456",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2 WHERE active = 1")
	require.NoError(t, err)

	require.Contains(t, output, "workspace_id = 'ws_456'")
	require.Contains(t, output, "active = 1")
	require.Contains(t, output, "AND")
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

		// Should only have workspace_id filter, no api_id filter
		require.Contains(t, result, "workspace_id = 'ws_test'")
		require.NotContains(t, result, "api_id IN")
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

		// Should have both workspace_id and key_space_id filters
		require.Contains(t, result, "workspace_id = 'ws_test'")
		require.Contains(t, result, "key_space_id IN ('ks_123')")
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

		// Should have both filters
		require.Contains(t, result, "workspace_id = 'ws_test'")
		// All three IDs should be in the IN clause
		require.Contains(t, result, "key_space_id IN")
		require.Contains(t, result, "'ks_123'")
		require.Contains(t, result, "'ks_456'")
		require.Contains(t, result, "'ks_789'")
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

		// Should combine all three filters with AND
		require.Contains(t, result, "workspace_id = 'ws_test'")
		require.Contains(t, result, "key_space_id IN ('ks_123')")
		require.Contains(t, result, "outcome = 'VALID'")
		require.Contains(t, result, "AND")
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
		require.Contains(t, result, "key_space_id IN")
		require.Contains(t, result, "key_space_id =")
		require.Contains(t, result, "'ks_123'") // Injected filter
		require.Contains(t, result, "'ks_999'") // User's filter
		require.Contains(t, result, "AND")      // Combined with AND
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

		// Should have workspace + both security filters
		require.Contains(t, result, "workspace_id = 'ws_test'")
		require.Contains(t, result, "key_space_id IN ('ks_123', 'ks_456')")
		require.Contains(t, result, "namespace_id IN ('nsid_111', 'nsid_222')")
		require.Contains(t, result, "AND")
	})
}
