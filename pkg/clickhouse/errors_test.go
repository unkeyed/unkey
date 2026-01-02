package clickhouse

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractUserFriendlyError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unknown identifier with suggestion",
			input:    `sendQuery: [HTTP 404] response body: "Code: 47. DB::Exception: Unknown expression identifier 'external_idd' in scope SELECT external_id, COUNT(*) AS total FROM default.key_verifications_raw_v2 WHERE (workspace_id = 'ws_4qD3194xe2x56qmv') AND (outcome = 'VALID') AND (time >= (now() - toIntervalDay(7))) GROUP BY external_idd LIMIT 10000000. Maybe you meant: ['external_id']. (UNKNOWN_IDENTIFIER) (version 25.6.4.12 (official build))\n"`,
			expected: "Unknown expression identifier 'external_idd' in scope SELECT external_id, COUNT(*) AS total FROM default.key_verifications_raw_v2 WHERE (workspace_id = 'ws_4qD3194xe2x56qmv') AND (outcome = 'VALID') AND (time >= (now() - toIntervalDay(7))) GROUP BY external_idd LIMIT 10000000. Maybe you meant: ['external_id']",
		},
		{
			name:     "syntax error",
			input:    `sendQuery: [HTTP 400] response body: "Code: 62. DB::Exception: Syntax error: failed at position 10. (SYNTAX_ERROR) (version 25.6.4.12)\n"`,
			expected: "Syntax error: failed at position 10",
		},
		{
			name:     "unknown table",
			input:    `Code: 60. DB::Exception: Table default.nonexistent doesn't exist. (UNKNOWN_TABLE) (version 25.6.4.12)`,
			expected: "Table default.nonexistent doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.input)
			result := ExtractUserFriendlyError(err)
			require.Equal(t, tt.expected, result)
		})
	}
}
