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

// TestClassifyPortalQueryErrorAdoptsResourceLimits guarantees the bounds
// withPortalQueryLimits sets reach the caller as rejections rather than
// internal errors. Without a code the API middleware answers 500, which tells a
// client to retry the query that just exceeded a limit.
func TestClassifyPortalQueryErrorAdoptsResourceLimits(t *testing.T) {
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
			name:      "memory",
			exception: &ch.Exception{Code: 241, Name: "DB::Exception", Message: "Memory limit (for query) exceeded"},
			expected:  codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		},
		{
			name:      "execution time",
			exception: &ch.Exception{Code: 159, Name: "DB::Exception", Message: "Timeout exceeded: elapsed 10.1 seconds, maximum: 10"},
			expected:  codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := fault.GetCode(classifyPortalQueryError(tt.exception))
			require.True(t, ok, "a resource limit must carry a code")
			require.Equal(t, tt.expected, code)
		})
	}
}

// TestClassifyPortalQueryErrorPreservesOperationalFailures guarantees anything
// that is not one of those bounds stays unclassified. These queries are written
// in this package rather than by the caller, so WrapClickHouseError's closing
// arm, which reads an unmatched failure as a bad customer query, would both
// blame the wrong party and hide an outage from the 5xx metrics.
func TestClassifyPortalQueryErrorPreservesOperationalFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "missing table",
			err:  &ch.Exception{Code: 60, Name: "DB::Exception", Message: "Table default.key_verifications_per_minute_v3 does not exist"},
		},
		{
			name: "connection refused",
			err:  errors.New("dial tcp 127.0.0.1:9000: connect: connection refused"),
		},
		{
			name: "closed connection",
			err:  errors.New("clickhouse: connection is closed"),
		},
		// The shared classifier matches "timeout" anywhere in a message, so a
		// transport failure reads as a query limit unless adoption is gated on
		// the ClickHouse exception itself.
		{
			name: "transport timeout",
			err:  errors.New("read tcp 10.0.0.2:52344->10.0.0.9:9000: i/o timeout"),
		},
		{
			name: "dial timeout",
			err:  errors.New("dial tcp 10.0.0.9:9000: i/o timeout"),
		},
		// Our own request deadline or a disconnected client, neither of which is
		// a limit the caller can do anything about.
		{
			name: "request context deadline",
			err:  context.DeadlineExceeded,
		},
		{
			name: "request cancelled",
			err:  context.Canceled,
		},
		// A ClickHouse exception that is not one of the limits we set.
		{
			name: "unrelated clickhouse exception",
			err:  &ch.Exception{Code: 202, Name: "DB::Exception", Message: "Too many simultaneous queries"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classified := classifyPortalQueryError(tt.err)
			require.Equal(t, tt.err, classified, "an operational failure must be returned untouched")

			_, ok := fault.GetCode(classified)
			require.False(t, ok, "an operational failure must not carry a user-facing code")
		})
	}
}
