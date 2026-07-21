package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser_BlockNonWhitelistedFunctions(t *testing.T) {
	p := newParserWithIdentityAliases(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name       string
		query      string
		shouldFail bool
	}{
		{
			name:       "file function",
			query:      "SELECT file('/etc/passwd') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "url function",
			query:      "SELECT url('http://evil.com/data') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "system function",
			query:      "SELECT system('rm -rf /') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "executable function",
			query:      "SELECT executable('/bin/bash') FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "dict_get to access dictionaries",
			query:      "SELECT dictGet('dict', 'attr', key_id) FROM default.key_verifications_raw_v2",
			shouldFail: true,
		},
		{
			name:       "nested safe functions should work",
			query:      "SELECT count(DISTINCT key_id) FROM default.key_verifications_raw_v2",
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)

			if tt.shouldFail {
				require.Error(t, err, "Dangerous function should be blocked")
				require.Contains(t, err.Error(), "not allowed", "Error should indicate function is not allowed")
			} else {
				require.NoError(t, err, "Safe function combination should work")
			}
		})
	}
}

func TestParser_AllowSafeFunctions(t *testing.T) {
	p := newParserWithIdentityAliases(Config{
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
	p := newParserWithIdentityAliases(Config{
		WorkspaceID: "ws_123",
		AllowedTables: []string{
			"default.key_verifications_raw_v2",
		},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "INSERT statement",
			query: "INSERT INTO default.key_verifications_raw_v2 (key_id) VALUES ('malicious')",
		},
		{
			name:  "UPDATE statement",
			query: "UPDATE default.key_verifications_raw_v2 SET key_id = 'hacked'",
		},
		{
			name:  "DELETE statement",
			query: "DELETE FROM default.key_verifications_raw_v2",
		},
		{
			name:  "DROP statement",
			query: "DROP TABLE default.key_verifications_raw_v2",
		},
		{
			name:  "CREATE statement",
			query: "CREATE TABLE malicious (id INT)",
		},
		{
			name:  "ALTER statement",
			query: "ALTER TABLE default.key_verifications_raw_v2 ADD COLUMN backdoor STRING",
		},
		{
			name:  "TRUNCATE statement",
			query: "TRUNCATE TABLE default.key_verifications_raw_v2",
		},
		{
			name:  "GRANT statement",
			query: "GRANT ALL ON *.* TO 'attacker'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			require.Error(t, err, "Only SELECT queries should be allowed")
		})
	}
}

// TestParser_RejectsSettingsClauses guarantees customer SQL cannot attach
// query-level settings to the root query or any nested SELECT query.
func TestParser_RejectsSettingsClauses(t *testing.T) {
	p := NewParser(Config{
		WorkspaceID:   "ws_123",
		AllowedTables: []string{"default.keys_v2"},
	})

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "top level",
			query: "SELECT * FROM default.keys_v2 SETTINGS max_execution_time = 0",
		},
		{
			name:  "scalar subquery",
			query: "SELECT (SELECT count(*) FROM default.keys_v2 SETTINGS max_execution_time = 0) FROM default.keys_v2",
		},
		{
			name:  "from subquery",
			query: "SELECT * FROM (SELECT * FROM default.keys_v2 SETTINGS max_execution_time = 0)",
		},
		{
			name:  "common table expression",
			query: "WITH cte AS (SELECT * FROM default.keys_v2 SETTINGS max_execution_time = 0) SELECT * FROM cte",
		},
		{
			name:  "union branch",
			query: "SELECT * FROM default.keys_v2 UNION ALL SELECT * FROM default.keys_v2 SETTINGS max_execution_time = 0",
		},
		{
			name:  "deeply nested",
			query: "SELECT * FROM (SELECT * FROM (SELECT * FROM default.keys_v2 SETTINGS max_execution_time = 0))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.Parse(context.Background(), tt.query)
			require.ErrorContains(t, err, "SETTINGS clauses are not allowed")
		})
	}
}
