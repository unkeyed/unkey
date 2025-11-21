package queryparser_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	chquery "github.com/unkeyed/unkey/go/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

func TestParser_WorkspaceFilter(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2")
	require.NoError(t, err)

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE workspace_id = 'ws_123'", output)
}

func TestParser_WorkspaceFilterWithExistingWhere(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
		WorkspaceID: "ws_456",
		AllowedTables: []string{
			"default.keys_v2",
		},
	})

	output, err := p.Parse(context.Background(), "SELECT * FROM default.keys_v2 WHERE active = 1")
	require.NoError(t, err)

	require.Equal(t, "SELECT * FROM default.keys_v2 WHERE workspace_id = 'ws_456' AND active = 1", output)
}

func TestSecurityFilterInjection(t *testing.T) {
	t.Run("no filter when SecurityFilters is empty", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
			WorkspaceID:     "ws_test",
			SecurityFilters: nil, // No restriction
			Limit:           100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_test' LIMIT 100", result)
	})

	t.Run("injects single key_space_id filter", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_test' AND key_space_id IN ('ks_123') LIMIT 100", result)
	})

	t.Run("injects multiple key_space_id filter", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123", "ks_456", "ks_789"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_test' AND key_space_id IN ('ks_123', 'ks_456', 'ks_789') LIMIT 100", result)
	})

	t.Run("combines with existing WHERE clause", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		query := "SELECT COUNT(*) FROM key_verifications WHERE outcome = 'VALID'"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_test' AND key_space_id IN ('ks_123') AND outcome = 'VALID' LIMIT 100", result)
	})

	t.Run("restricts access even when user queries different key_space_id", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123"}, // User only has access to ks_123
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"key_verifications": "default.key_verifications_raw_v2",
			},
			AllowedTables: []string{
				"default.key_verifications_raw_v2",
			},
		})

		// User tries to query ks_999 which they don't have access to
		query := "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_999'"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		// Both filters are present, creating impossible AND condition
		// Injected: key_space_id IN ('ks_123') - only ks_123
		// User's: key_space_id = 'ks_999'
		// Result: no rows (ks_123 AND ks_999 = impossible)
		require.Equal(t, "SELECT COUNT(*) FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_test' AND key_space_id IN ('ks_123') AND key_space_id = 'ks_999' LIMIT 100", result)
	})

	t.Run("supports multiple security filters simultaneously", func(t *testing.T) {
		parser := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
			WorkspaceID: "ws_test",
			SecurityFilters: []chquery.SecurityFilter{
				{
					Column:        "key_space_id",
					AllowedValues: []string{"ks_123", "ks_456"},
				},
				{
					Column:        "namespace_id",
					AllowedValues: []string{"nsid_111", "nsid_222"},
				},
			},
			Limit: 100,
			TableAliases: map[string]string{
				"ratelimits": "default.ratelimits_v2",
			},
			AllowedTables: []string{
				"default.ratelimits_v2",
			},
		})

		query := "SELECT COUNT(*) FROM ratelimits"
		result, err := parser.Parse(context.Background(), query)
		require.NoError(t, err)

		require.Equal(t, "SELECT COUNT(*) FROM default.ratelimits_v2 WHERE workspace_id = 'ws_test' AND namespace_id IN ('nsid_111', 'nsid_222') AND key_space_id IN ('ks_123', 'ks_456') LIMIT 100", result)
	})
}

func TestParser_WorkspaceFilterInjection(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
		WorkspaceID: "ws_victim",
		Limit:       1000,
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
			name:     "OR to bypass workspace filter",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_attacker' OR 1=1",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_victim' AND workspace_id = 'ws_attacker' OR 1 = 1 LIMIT 1000",
		},
		{
			name:     "NOT to invert workspace filter",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE NOT workspace_id = 'ws_victim'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_victim' AND NOT workspace_id = 'ws_victim' LIMIT 1000",
		},
		{
			name:     "workspace_id in SELECT to confuse parser",
			query:    "SELECT workspace_id FROM default.key_verifications_raw_v2 WHERE key_id = 'test'",
			expected: "SELECT workspace_id FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_victim' AND key_id = 'test' LIMIT 1000",
		},
		{
			name:     "workspace_id with different case",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE WORKSPACE_ID = 'ws_attacker'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_victim' AND WORKSPACE_ID = 'ws_attacker' LIMIT 1000",
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

func TestParser_SQLInjectionWithFilters(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
		WorkspaceID: "ws_123",
		Limit:       1000,
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
			name:     "injection in WHERE clause with quotes",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '' OR '1'='1'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND key_id = '' OR '1' = '1' LIMIT 1000",
		},
		{
			name:     "injection with comment",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '' -- comment",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND key_id = '' LIMIT 1000",
		},
		{
			name:     "injection with multiline comment",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '/* comment */'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND key_id = '/* comment */' LIMIT 1000",
		},
		{
			name:     "injection with semicolon",
			query:    "SELECT * FROM default.key_verifications_raw_v2; DROP TABLE users",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' LIMIT 1000",
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

func TestParser_SpecialCharactersInFilters(t *testing.T) {
	p := chquery.NewParser(chquery.Config{
		Logger: logging.NewNoop(),
		WorkspaceID: "ws_123",
		Limit:       1000,
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
			name:     "null bytes",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '\x00'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND key_id = '\x00' LIMIT 1000",
		},
		{
			name:     "unicode characters",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '你好'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND key_id = '你好' LIMIT 1000",
		},
		{
			name:     "emoji",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '🔥'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND key_id = '🔥' LIMIT 1000",
		},
		{
			name:     "backslashes",
			query:    "SELECT * FROM default.key_verifications_raw_v2 WHERE key_id = '\\\\'",
			expected: "SELECT * FROM default.key_verifications_raw_v2 WHERE workspace_id = 'ws_123' AND key_id = '\\\\' LIMIT 1000",
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
