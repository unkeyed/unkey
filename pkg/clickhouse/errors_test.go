package clickhouse

import (
	"context"
	"errors"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
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
			expected: "Unknown identifier in analytics query",
		},
		{
			name:     "syntax error",
			input:    `sendQuery: [HTTP 400] response body: "Code: 62. DB::Exception: Syntax error: failed at position 10. (SYNTAX_ERROR) (version 25.6.4.12)\n"`,
			expected: "Invalid SQL syntax",
		},
		{
			name:     "unknown table",
			input:    `Code: 60. DB::Exception: Table default.nonexistent doesn't exist. (UNKNOWN_TABLE) (version 25.6.4.12)`,
			expected: "Invalid analytics query",
		},
		// ClickHouse answers this for SELECT * on a table with a column grant,
		// because it expands the star to every physical column.
		{
			name:     "column outside the grant",
			input:    `code: 497, message: ws_test: Not enough privileges. To execute this query, it's necessary to have the grant SELECT(path, platform) ON default.frontline_requests_raw_v1. (Missing permissions: SELECT(platform) ON default.frontline_requests_raw_v1)`,
			expected: "The query reads a column that is not available. Select only the documented columns instead of *",
		},
		// The Go driver answers this for a percentile column of a rollup table.
		{
			name:     "aggregate state column",
			input:    `read data: failed to decode block: clickhouse: unsupported column type "AggregateFunction(quantileTDigest(0.5), Float64)"`,
			expected: "The query selects an aggregate state column. Use quantileTDigestMerge to read a percentile from it",
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

// Security guarantee: ClickHouse result limits map to stable resource errors rather than timeout or 500 responses.
func TestWrapClickHouseError_ResultLimits(t *testing.T) {
	tests := []struct {
		name      string
		exception *ch.Exception
		expected  codes.URN
	}{
		{
			name:      "result bytes",
			exception: &ch.Exception{Code: 396, Name: "DB::Exception", Message: "Limit for result exceeded, max bytes: 4194304"},
			expected:  codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		},
		{
			name:      "result rows",
			exception: &ch.Exception{Code: 396, Name: "DB::Exception", Message: "Limit for result exceeded, max rows: 10000"},
			expected:  codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		},
		{
			name:      "legacy result rows code",
			exception: &ch.Exception{Code: 158, Name: "DB::Exception", Message: "Too many rows"},
			expected:  codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		},
		{
			name:      "query cancelled",
			exception: &ch.Exception{Code: 394, Name: "DB::Exception", Message: "Query was cancelled"},
			expected:  codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		},
		{
			name:      "result rows by exception name",
			exception: &ch.Exception{Code: 999, Name: "TOO_MANY_ROWS", Message: "Result limit exceeded"},
			expected:  codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		},
		{
			name:      "query cancelled by exception name",
			exception: &ch.Exception{Code: 999, Name: "QUERY_WAS_CANCELLED", Message: "Query stopped"},
			expected:  codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapClickHouseError(tt.exception)
			code, ok := fault.GetCode(err)
			require.True(t, ok)
			require.Equal(t, tt.expected, code)
		})
	}
}

// TestWrapClickHouseError_HidesRewrittenQuery guarantees public errors do not
// reveal physical tables or workspace predicates added during query rewriting.
func TestWrapClickHouseError_HidesRewrittenQuery(t *testing.T) {
	workspaceID := "ws_private_123"
	err := errors.New("Unknown expression identifier `missing` in scope SELECT missing FROM default.key_verifications_raw_v2 WHERE key_verifications_raw_v2.workspace_id = '" + workspaceID + "'")

	publicMessage := fault.UserFacingMessage(WrapClickHouseError(err))
	require.NotContains(t, publicMessage, "default.key_verifications_raw_v2")
	require.NotContains(t, publicMessage, workspaceID)
	require.NotContains(t, publicMessage, "workspace_id")
}

// TestWrapClickHouseError_ContextCancellation guarantees API cancellation is
// classified as an execution timeout rather than an invalid customer query.
func TestWrapClickHouseError_ContextCancellation(t *testing.T) {
	for _, contextErr := range []error{context.Canceled, context.DeadlineExceeded} {
		err := WrapClickHouseError(contextErr)
		code, ok := fault.GetCode(err)
		require.True(t, ok)
		require.Equal(t, codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(), code)
		require.ErrorIs(t, err, contextErr)
	}
}
