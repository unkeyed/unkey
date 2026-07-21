package queryparser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExtractColumnValues guarantees authorization-relevant literals are
// found across qualified predicates, nested queries, and supported operators.
func TestExtractColumnValues(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		columnName string
		expected   []string
	}{
		{
			name:       "single equality",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_1234'",
			columnName: "key_space_id",
			expected:   []string{"ks_1234"},
		},
		{
			name:       "IN clause with multiple values",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id IN ('ks_1234', 'ks_5678')",
			columnName: "key_space_id",
			expected:   []string{"ks_1234", "ks_5678"},
		},
		{
			name:       "qualified equality",
			query:      "SELECT COUNT(*) FROM key_verifications AS r WHERE r.key_space_id = 'ks_qualified'",
			columnName: "key_space_id",
			expected:   []string{"ks_qualified"},
		},
		{
			name:       "qualified IN",
			query:      "SELECT COUNT(*) FROM key_verifications AS r WHERE r.key_space_id IN ('ks_first', 'ks_second')",
			columnName: "key_space_id",
			expected:   []string{"ks_first", "ks_second"},
		},
		{
			name:       "multiple conditions with AND",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_1234' AND outcome = 'VALID'",
			columnName: "key_space_id",
			expected:   []string{"ks_1234"},
		},
		{
			name:       "no matching column",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE outcome = 'VALID'",
			columnName: "key_space_id",
			expected:   []string{},
		},
		{
			name:       "case insensitive column matching",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE KEY_SPACE_ID = 'ks_1234'",
			columnName: "key_space_id",
			expected:   []string{"ks_1234"},
		},
		{
			name:       "OR expression",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_1234' OR key_space_id = 'ks_5678'",
			columnName: "key_space_id",
			expected:   []string{"ks_1234", "ks_5678"},
		},
		{
			name:       "complex query with multiple operators",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_1234' AND (outcome = 'VALID' OR outcome = 'INVALID')",
			columnName: "key_space_id",
			expected:   []string{"ks_1234"},
		},
		{
			name:       "extract different column",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_1234' AND outcome = 'VALID'",
			columnName: "outcome",
			expected:   []string{"VALID"},
		},
		{
			name:       "no WHERE clause",
			query:      "SELECT COUNT(*) FROM key_verifications",
			columnName: "key_space_id",
			expected:   []string{},
		},
		{
			name:       "duplicate values deduplicated",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_1234' OR key_space_id IN ('ks_5678', 'ks_1234')",
			columnName: "key_space_id",
			expected:   []string{"ks_1234", "ks_5678"},
		},
		{
			name:       "HAVING clause",
			query:      "SELECT key_space_id, COUNT(*) FROM key_verifications GROUP BY key_space_id HAVING key_space_id = 'ks_1234'",
			columnName: "key_space_id",
			expected:   []string{"ks_1234"},
		},
		{
			name:       "nested branches retain first-seen order",
			query:      "SELECT * FROM key_verifications WHERE key_space_id = 'ks_outer' AND key_id IN (SELECT key_id FROM key_verifications WHERE key_space_id IN ('ks_inner_1', 'ks_inner_2'))",
			columnName: "key_space_id",
			expected:   []string{"ks_outer", "ks_inner_1", "ks_inner_2"},
		},
		{
			name:       "EXCEPT branches retain first-seen order",
			query:      "SELECT * FROM key_verifications WHERE key_space_id = 'ks_left' EXCEPT SELECT * FROM key_verifications WHERE key_space_id = 'ks_right'",
			columnName: "key_space_id",
			expected:   []string{"ks_left", "ks_right"},
		},
		{
			name:       "negative operator != ignored",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id != 'ks_bad'",
			columnName: "key_space_id",
			expected:   []string{},
		},
		{
			name:       "mix of positive and negative operators",
			query:      "SELECT COUNT(*) FROM key_verifications WHERE key_space_id = 'ks_good' AND outcome != 'INVALID'",
			columnName: "key_space_id",
			expected:   []string{"ks_good"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(Config{
				WorkspaceID:   "ws_test",
				TableAliases:  map[string]string{"key_verifications": "default.key_verifications_raw_v2"},
				AllowedTables: []string{"default.key_verifications_raw_v2"},
			})

			// Parse the query first
			_, err := parser.Parse(context.Background(), tt.query)
			require.NoError(t, err)

			// Extract values
			values := parser.ExtractColumn(tt.columnName)

			require.Equal(t, tt.expected, values)
		})
	}
}
