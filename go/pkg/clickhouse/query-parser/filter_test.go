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

	require.Contains(t, output.Query, "workspace_id = 'ws_123'")
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

	require.Contains(t, output.Query, "workspace_id = 'ws_456'")
	require.Contains(t, output.Query, "active = 1")
	require.Contains(t, output.Query, "AND")
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
		require.Contains(t, result.Query, "workspace_id = 'ws_test'")
		require.NotContains(t, result.Query, "api_id IN")
	})

	t.Run("injects single API ID filter", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "api_id",
					AllowedValues: []string{"api_123"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
			VirtualColumns: map[string]chquery.VirtualColumn{
				"apiId": {
					ActualColumn: "key_space_id",
					Aliases:      []string{"api_id"},
					Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
						return map[string]string{"api_123": "ks_abc"}, nil
					},
				},
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should have both workspace_id and api_id filters
		require.Contains(t, result.Query, "workspace_id = 'ws_test'")
		// Virtual column should be resolved: api_id → key_space_id, 'api_123' → 'ks_abc'
		require.Contains(t, result.Query, "key_space_id IN ('ks_abc')")
	})

	t.Run("injects multiple API IDs filter", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "api_id",
					AllowedValues: []string{"api_123", "api_456", "api_789"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
			VirtualColumns: map[string]chquery.VirtualColumn{
				"apiId": {
					ActualColumn: "key_space_id",
					Aliases:      []string{"api_id"},
					Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
						return map[string]string{
							"api_123": "ks_abc",
							"api_456": "ks_def",
							"api_789": "ks_ghi",
						}, nil
					},
				},
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should have both filters
		require.Contains(t, result.Query, "workspace_id = 'ws_test'")
		// All three resolved IDs should be in the IN clause
		require.Contains(t, result.Query, "key_space_id IN")
		require.Contains(t, result.Query, "'ks_abc'")
		require.Contains(t, result.Query, "'ks_def'")
		require.Contains(t, result.Query, "'ks_ghi'")
	})

	t.Run("combines with existing WHERE clause", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "api_id",
					AllowedValues: []string{"api_123"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
			VirtualColumns: map[string]chquery.VirtualColumn{
				"apiId": {
					ActualColumn: "key_space_id",
					Aliases:      []string{"api_id"},
					Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
						return map[string]string{"api_123": "ks_abc"}, nil
					},
				},
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications WHERE outcome = 'VALID'"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should combine all three filters with AND
		require.Contains(t, result.Query, "workspace_id = 'ws_test'")
		require.Contains(t, result.Query, "key_space_id IN ('ks_abc')")
		require.Contains(t, result.Query, "outcome = 'VALID'")
		require.Contains(t, result.Query, "AND")
	})

	t.Run("restricts access even when user queries different API", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "api_id",
					AllowedValues: []string{"api_123"}, // User only has access to api_123
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
			VirtualColumns: map[string]chquery.VirtualColumn{
				"apiId": {
					ActualColumn: "key_space_id",
					Aliases:      []string{"api_id"},
					Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
						return map[string]string{
							"api_123": "ks_abc",
							"api_999": "ks_xyz", // Different API
						}, nil
					},
				},
			},
		})

		// User tries to query api_999 which they don't have access to
		query := "SELECT COUNT(*) FROM key_verifications WHERE api_id = 'api_999'"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Both filters are present, creating impossible AND condition
		// Injected: key_space_id IN ('ks_abc') - only api_123
		// User's: key_space_id = 'ks_xyz' - api_999
		// Result: no rows (ks_abc AND ks_xyz = impossible)
		require.Contains(t, result.Query, "key_space_id IN")
		require.Contains(t, result.Query, "key_space_id =")
		require.Contains(t, result.Query, "'ks_abc'") // Injected filter
		require.Contains(t, result.Query, "'ks_xyz'") // User's filter
		require.Contains(t, result.Query, "AND")      // Combined with AND
	})

	t.Run("supports multiple security filters simultaneously", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "api_id",
					AllowedValues: []string{"api_123", "api_456"},
				},
				{
					Column:        "namespace",
					AllowedValues: []string{"ns_prod", "ns_staging"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"ratelimits": "default.ratelimits_v2",
			},
			AllowedTables: []string{
				"default.ratelimits_v2",
			},
			VirtualColumns: map[string]chquery.VirtualColumn{
				"apiId": {
					ActualColumn: "key_space_id",
					Aliases:      []string{"api_id"},
					Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
						return map[string]string{
							"api_123": "ks_abc",
							"api_456": "ks_def",
						}, nil
					},
				},
				"namespace": {
					ActualColumn: "namespace_id",
					Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
						return map[string]string{
							"ns_prod":    "nsid_111",
							"ns_staging": "nsid_222",
						}, nil
					},
				},
			},
		})

		query := "SELECT COUNT(*) FROM ratelimits"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should have workspace + both security filters
		require.Contains(t, result.Query, "workspace_id = 'ws_test'")
		require.Contains(t, result.Query, "key_space_id IN ('ks_abc', 'ks_def')")
		require.Contains(t, result.Query, "namespace_id IN ('nsid_111', 'nsid_222')")
		require.Contains(t, result.Query, "AND")
	})
}
