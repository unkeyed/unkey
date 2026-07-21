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

// Security guarantee: parser work is bounded before attacker-controlled SQL reaches the lexer.
func TestParser_RejectsQueriesOverByteLimit(t *testing.T) {
	p := NewParser(Config{QueryBytesMax: 32})

	_, err := p.Parse(context.Background(), "SELECT "+strings.Repeat("1 + ", 20)+"1")
	require.ErrorContains(t, err, "query exceeds maximum length")
}

// Security guarantee: one row cannot bypass row caps with an unbounded number of projected values.
func TestParser_RejectsWideProjections(t *testing.T) {
	p := NewParser(Config{ProjectedColumnsMax: 3})

	_, err := p.Parse(context.Background(), "SELECT 1, 2, 3, 4")
	require.ErrorContains(t, err, "too many projected columns")
}

// Security guarantee: projection limits include EXCEPT branches omitted by the dependency walker.
func TestParser_RejectsWideExceptProjection(t *testing.T) {
	p := NewParser(Config{ProjectedColumnsMax: 3})

	_, err := p.Parse(context.Background(), "SELECT 1 EXCEPT (SELECT 1, 2, 3, 4)")
	require.ErrorContains(t, err, "too many projected columns")
}

// Security guarantee: short but deeply composed SQL cannot consume unbounded parser or rewrite CPU.
func TestParser_RejectsComplexAST(t *testing.T) {
	p := NewParser(Config{ASTNodesMax: 10})

	_, err := p.Parse(context.Background(), "SELECT 1 + 2 + 3 + 4 + 5 + 6")
	require.ErrorContains(t, err, "query is too complex")
}
