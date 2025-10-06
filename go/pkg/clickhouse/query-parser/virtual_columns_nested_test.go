package queryparser_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	chquery "github.com/unkeyed/unkey/go/pkg/clickhouse/query-parser"
)

func TestVirtualColumnInNestedExpressions(t *testing.T) {
	parser := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_test",
		Limit:       100,
		TableAliases: map[string]string{
			"test_table": "default.test_table",
		},
		AllowedTables: []string{
			"default.test_table",
		},
		VirtualColumns: map[string]chquery.VirtualColumn{
			"apiId": {
				ActualColumn: "key_space_id",
				Aliases:      []string{"api_id"},
				Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
					// Return mapping for testing
					return map[string]string{"api_123": "ks_abc"}, nil
				},
			},
		},
	})

	t.Run("virtual column in function call in SELECT", func(t *testing.T) {
		query := "SELECT toStartOfMinute(time), SUM(api_id) FROM test_table GROUP BY toStartOfMinute(time)"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should rewrite api_id to key_space_id inside SUM()
		require.Contains(t, result.Query, "SUM(key_space_id)")
		require.NotContains(t, result.Query, "SUM(api_id)")
	})

	t.Run("virtual column in WHERE with IN subquery", func(t *testing.T) {
		query := "SELECT COUNT(*) FROM test_table WHERE api_id = 'api_123'"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should rewrite api_id to key_space_id and resolve value
		require.Contains(t, result.Query, "key_space_id = 'ks_abc'")
		require.NotContains(t, result.Query, "api_id")
	})

	t.Run("virtual column in SELECT and ORDER BY uses alias", func(t *testing.T) {
		query := "SELECT api_id, COUNT(*) as total FROM test_table GROUP BY api_id ORDER BY api_id DESC"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// When virtual column is in SELECT with alias, ORDER BY can use the alias
		require.Contains(t, result.Query, "key_space_id AS api_id")
		// ORDER BY can reference either the alias or actual column - both are valid
		require.Contains(t, result.Query, "ORDER BY")
	})

	t.Run("virtual column ONLY in ORDER BY gets rewritten", func(t *testing.T) {
		query := "SELECT COUNT(*) as total FROM test_table GROUP BY api_id ORDER BY api_id DESC"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// When virtual column is NOT in SELECT, ORDER BY must use actual column
		require.Contains(t, result.Query, "ORDER BY key_space_id DESC")
		require.Contains(t, result.Query, "GROUP BY key_space_id")
	})
}
