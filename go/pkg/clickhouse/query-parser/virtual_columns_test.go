package queryparser

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_VirtualColumns(t *testing.T) {
	mockResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, id := range virtualIDs {
			result[id] = "actual_" + id
		}
		return result, nil
	}

	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId": {
				ActualColumn: "key_space_id",
				Resolver:     mockResolver,
			},
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2 WHERE apiId = 'api_123'")
	require.NoError(t, err)

	require.Contains(t, output, "key_space_id")
	require.Contains(t, output, "actual_api_123")
}

func TestParser_VirtualColumnsWithAliases(t *testing.T) {
	mockResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, id := range virtualIDs {
			result[id] = "actual_" + id
		}
		return result, nil
	}

	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId": {
				ActualColumn: "key_space_id",
				Aliases:      []string{"api_id"},
				Resolver:     mockResolver,
			},
		},
	})

	// Test with canonical name
	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2 WHERE apiId = 'api_123'")
	require.NoError(t, err)
	require.Contains(t, output, "key_space_id")
	require.Contains(t, output, "actual_api_123")

	// Test with alias
	output, err = p.Parse(context.Background(), "SELECT * FROM default.keys_v2 WHERE api_id = 'api_456'")
	require.NoError(t, err)
	require.Contains(t, output, "key_space_id")
	require.Contains(t, output, "actual_api_456")
}

func TestParser_VirtualColumnsWithIN(t *testing.T) {
	mockResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, id := range virtualIDs {
			result[id] = "actual_" + id
		}
		return result, nil
	}

	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId": {
				ActualColumn: "key_space_id",
				Resolver:     mockResolver,
			},
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2 WHERE apiId IN ('api_1', 'api_2', 'api_3')")
	require.NoError(t, err)

	require.Contains(t, output, "key_space_id")
	require.Contains(t, output, "actual_api_1")
	require.Contains(t, output, "actual_api_2")
	require.Contains(t, output, "actual_api_3")
}

func TestParser_VirtualColumnsWithDifferentOperators(t *testing.T) {
	mockResolver := func(ctx context.Context, virtualIDs []string) (map[string]string, error) {
		result := make(map[string]string)
		for _, id := range virtualIDs {
			result[id] = "actual_" + id
		}
		return result, nil
	}

	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
		VirtualColumns: map[string]VirtualColumn{
			"apiId": {
				ActualColumn: "key_space_id",
				Resolver:     mockResolver,
			},
		},
	})

	operators := []string{"=", "!=", "<", ">", "<=", ">="}
	for _, op := range operators {
		query := fmt.Sprintf("SELECT * FROM default.keys_v2 WHERE apiId %s 'api_123'", op)
		output, err := p.Parse(context.Background(), query)
		require.NoError(t, err, "Failed with operator: %s", op)
		require.Contains(t, output, "key_space_id", "Column name not rewritten for operator: %s", op)
		require.Contains(t, output, "actual_api_123", "Value not resolved for operator: %s", op)
	}
}
