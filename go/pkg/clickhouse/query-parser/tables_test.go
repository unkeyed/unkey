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

	require.Contains(t, output.Query, "keys_v2")
}

func TestParser_BlockSystemTables(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
	})

	_, err := p.Parse(context.Background(), "SELECT * FROM system.tables")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
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
