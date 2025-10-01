package query

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewriter_BasicSelect(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	input := "SELECT * FROM key_verifications"
	output, err := r.Rewrite(context.Background(), input)
	require.NoError(t, err)

	// Should inject workspace_id filter and resolve alias
	require.Contains(t, output, "workspace_id = 'ws_123'")
	require.Contains(t, output, "key_verifications_v1")
}

func TestRewriter_WithExistingWhere(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	input := "SELECT * FROM key_verifications WHERE valid = true"
	output, err := r.Rewrite(context.Background(), input)
	require.NoError(t, err)

	// Should inject workspace_id AND combine with existing filter
	require.Contains(t, output, "workspace_id = 'ws_123'")
	require.Contains(t, output, "valid = true")
	require.Contains(t, strings.ToLower(output), "and")
}

func TestRewriter_BlockInsert(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
	})

	input := "INSERT INTO key_verifications VALUES (1, 2, 3)"
	_, err := r.Rewrite(context.Background(), input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only SELECT queries are allowed")
}

func TestRewriter_BlockUpdate(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
	})

	input := "UPDATE key_verifications SET valid = false"
	_, err := r.Rewrite(context.Background(), input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only SELECT queries are allowed")
}

func TestRewriter_BlockDelete(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
	})

	input := "DELETE FROM key_verifications WHERE id = 1"
	_, err := r.Rewrite(context.Background(), input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only SELECT queries are allowed")
}

func TestRewriter_BlockSystemTables(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
	})

	tests := []string{
		"SELECT * FROM system.tables",
		"SELECT * FROM system.databases",
		"SELECT * FROM information_schema.tables",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := r.Rewrite(context.Background(), input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not allowed")
		})
	}
}

func TestRewriter_BlockDangerousFunctions(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []string{
		"SELECT file('/etc/passwd') FROM key_verifications",
		"SELECT url('http://evil.com') FROM key_verifications",
		"SELECT executable('rm -rf /') FROM key_verifications",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := r.Rewrite(context.Background(), input)
			require.Error(t, err)
			require.Contains(t, err.Error(), "not allowed")
		})
	}
}

func TestRewriter_AllowAggregateFunctions(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	tests := []string{
		"SELECT COUNT(*) FROM key_verifications",
		"SELECT SUM(count) FROM key_verifications",
		"SELECT AVG(latency) FROM key_verifications",
		"SELECT MAX(timestamp) FROM key_verifications",
		"SELECT MIN(timestamp) FROM key_verifications",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			output, err := r.Rewrite(context.Background(), input)
			require.NoError(t, err)
			require.Contains(t, output, "workspace_id = 'ws_123'")
		})
	}
}

func TestRewriter_AllowJoins(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
			"ratelimits":        "default.ratelimits_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
			"default.ratelimits_v1",
		},
	})

	input := "SELECT * FROM key_verifications JOIN ratelimits ON key_verifications.key_id = ratelimits.key_id"
	output, err := r.Rewrite(context.Background(), input)
	require.NoError(t, err)

	// Should inject workspace_id and resolve both table aliases
	require.Contains(t, output, "workspace_id = 'ws_123'")
	require.Contains(t, output, "key_verifications_v1")
	require.Contains(t, output, "ratelimits_v1")
}

func TestRewriter_BlockUnallowedTable(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"key_verifications_v1",
		},
	})

	input := "SELECT * FROM other_table"
	_, err := r.Rewrite(context.Background(), input)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestRewriter_GroupByAndOrderBy(t *testing.T) {
	r := New(Config{
		WorkspaceID: "ws_123",
		TableAliases: map[string]string{
			"key_verifications": "default.key_verifications_v1",
		},
		AllowedTables: []string{
			"default.key_verifications_v1",
		},
	})

	input := "SELECT key_id, COUNT(*) as count FROM key_verifications GROUP BY key_id ORDER BY count DESC LIMIT 10"
	output, err := r.Rewrite(context.Background(), input)
	require.NoError(t, err)

	require.Contains(t, output, "workspace_id = 'ws_123'")
	require.Contains(t, strings.ToLower(output), "group by")
	require.Contains(t, strings.ToLower(output), "order by")
	require.Contains(t, strings.ToLower(output), "limit")
}
