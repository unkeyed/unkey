package queryparser_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	chquery "github.com/unkeyed/unkey/go/pkg/clickhouse/query-parser"
)

func TestVirtualColumnInGroupBy(t *testing.T) {
	parser := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_test",
		Limit:       100,
		VirtualColumns: map[string]chquery.VirtualColumn{
			"externalId": {
				ActualColumn: "identity_id",
				Aliases:      []string{"external_id"},
				Resolver: func(ctx context.Context, externalIDs []string) (map[string]string, error) {
					return map[string]string{"ext1": "id1"}, nil
				},
			},
		},
	})

	query := "SELECT COUNT(*) FROM table GROUP BY externalId"
	result, err := parser.Parse(context.Background(), query)
	require.NoError(t, err)

	// Should rewrite externalId to identity_id in GROUP BY
	require.Contains(t, result.Query, "GROUP BY identity_id")
	require.NotContains(t, result.Query, "externalId")
}

func TestVirtualColumnInSelect(t *testing.T) {
	parser := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_test",
		Limit:       100,
		VirtualColumns: map[string]chquery.VirtualColumn{
			"externalId": {
				ActualColumn: "identity_id",
				Aliases:      []string{"external_id"},
				Resolver: func(ctx context.Context, externalIDs []string) (map[string]string, error) {
					return map[string]string{"ext1": "id1"}, nil
				},
			},
		},
	})

	t.Run("canonical name preserves column name", func(t *testing.T) {
		query := "SELECT externalId, COUNT(*) FROM table GROUP BY externalId"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should rewrite externalId to identity_id but add AS externalId to preserve column name
		require.Contains(t, result.Query, "identity_id AS externalId")
		require.Contains(t, result.Query, "GROUP BY identity_id")
	})

	t.Run("alias preserves original alias name", func(t *testing.T) {
		query := "SELECT external_id, COUNT(*) FROM table GROUP BY external_id"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Should rewrite external_id to identity_id but add AS external_id to preserve column name
		require.Contains(t, result.Query, "identity_id AS external_id")
		require.Contains(t, result.Query, "GROUP BY identity_id")
	})

	t.Run("user-defined alias is preserved", func(t *testing.T) {
		query := "SELECT externalId AS my_alias FROM table"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// User already provided alias, should not add another
		require.Contains(t, result.Query, "identity_id AS my_alias")
		require.NotContains(t, result.Query, "AS externalId")
	})
}

func TestColumnMappingsOnlyForVirtualColumns(t *testing.T) {
	parser := chquery.NewParser(chquery.Config{
		WorkspaceID: "ws_test",
		Limit:       100,
		VirtualColumns: map[string]chquery.VirtualColumn{
			"apiId": {
				ActualColumn: "key_space_id",
				Aliases:      []string{"api_id"},
				Resolver: func(ctx context.Context, ids []string) (map[string]string, error) {
					return map[string]string{}, nil
				},
			},
		},
	})

	query := "SELECT outcome, api_id, key_id, sum(count) as total FROM table GROUP BY outcome, api_id, key_id"
	result, err := parser.Parse(context.Background(), query)
	require.NoError(t, err)

	// Only api_id should have a mapping (virtual column)
	// outcome, key_id, and total should NOT have mappings (regular columns)
	require.Len(t, result.ColumnMappings, 1, "should only have mapping for virtual column")

	// Verify the one mapping is for api_id -> key_space_id
	require.Equal(t, "api_id", result.ColumnMappings[0].ResultColumn)
	require.Equal(t, "key_space_id", result.ColumnMappings[0].ActualColumn)
}
