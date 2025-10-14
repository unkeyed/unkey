package queryparser

import (
	"context"
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

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE workspace_id = 'ws_123' LIMIT 100", output)
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

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE workspace_id = 'ws_123' LIMIT 50", output)
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

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE workspace_id = 'ws_123' LIMIT 10", output)
}

func TestParser_LimitBypassAttempts(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID: "ws_123",
		Limit:       10,
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
			name:     "LIMIT with OFFSET to read more",
			query:    "SELECT * FROM default.key_verifications_raw_v2 LIMIT 100000 OFFSET 0",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' LIMIT 10 OFFSET 0",
		},
		{
			name:     "extremely high LIMIT",
			query:    "SELECT * FROM default.key_verifications_raw_v2 LIMIT 999999999",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' LIMIT 10",
		},
		{
			name:     "negative LIMIT",
			query:    "SELECT * FROM default.key_verifications_raw_v2 LIMIT -1",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' LIMIT 10",
		},
		{
			name:     "LIMIT ALL",
			query:    "SELECT * FROM default.key_verifications_raw_v2 LIMIT ALL",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' LIMIT 10",
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
