package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_WorkspaceFilter(t *testing.T) {
	p := NewParser(Config{
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
	p := NewParser(Config{
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
