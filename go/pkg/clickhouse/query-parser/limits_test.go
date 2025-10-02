package queryparser

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_EnforceLimit(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       100,
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2 LIMIT 1000")
	require.NoError(t, err)

	require.Contains(t, strings.ToLower(output), "limit 100")
}

func TestParser_AddLimit(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       50,
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2")
	require.NoError(t, err)

	require.Contains(t, strings.ToLower(output), "limit 50")
}

func TestParser_PreserveSmallerLimit(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       100,
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2 LIMIT 10")
	require.NoError(t, err)

	require.Contains(t, strings.ToLower(output), "limit 10")
}
