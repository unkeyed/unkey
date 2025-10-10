package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_BlockNonWhitelistedFunctions(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	dangerousFuncs := []string{"file", "url", "executable", "system", "shell", "pipe", "readfile", "mysql"}

	for _, fn := range dangerousFuncs {
		query := "SELECT " + fn + "('test') FROM default.keys_v2"
		_, err := p.Parse(context.Background(), query)
		require.Error(t, err, "Function %s should be blocked", fn)
		require.Contains(t, err.Error(), "not allowed")
	}
}

func TestParser_AllowSafeFunctions(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	safeFuncs := []string{"count", "sum", "avg", "max", "min", "now", "toDate"}

	for _, fn := range safeFuncs {
		query := "SELECT " + fn + "(*) FROM default.keys_v2"
		_, err := p.Parse(context.Background(), query)
		require.NoError(t, err, "Function %s should be allowed", fn)
	}
}

func TestParser_OnlySelectAllowed(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	_, err := p.Parse(context.Background(), "INSERT INTO default.keys_v2 VALUES (1)")
	require.Error(t, err)
	require.Contains(t, err.Error(), "SELECT")
}
