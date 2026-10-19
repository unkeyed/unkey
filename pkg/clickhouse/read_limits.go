package clickhouse

import (
	"context"
	"errors"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

const (
	// readExecutionTimeSecondsMax bounds how long one read may occupy the
	// shared connection.
	readExecutionTimeSecondsMax = 10
	// readMemoryBytesMax bounds one read's server-side memory.
	readMemoryBytesMax = 1 << 30
)

// withReadLimits bounds a single fixed-shape read that runs on the connection
// every workspace shares, such as a portal timeseries or a log listing. The
// operator analytics routes get the equivalent from a per-workspace ClickHouse
// settings profile, which this path has no user to hang off, so the bounds
// ride on the query instead.
//
// max_result_bytes stops an oversized read server-side, before the rows are
// shipped.
//
// use_with_fill_by_sorting_prefix keeps each key_id series of the per-key
// verification timeseries contiguous. It defaults on, but a settings profile on
// this shared connection could turn it off.
func withReadLimits(ctx context.Context) context.Context {
	return ch.Context(ctx, ch.WithSettings(ch.Settings{
		"max_execution_time":              readExecutionTimeSecondsMax,
		"max_memory_usage":                readMemoryBytesMax,
		"max_result_bytes":                AnalyticsResultBytesMax,
		"result_overflow_mode":            "throw",
		"use_with_fill_by_sorting_prefix": 1,
	}))
}

// readLimitCodes are the ClickHouse exceptions the resource limits in
// [withReadLimits] can raise: TIMEOUT_EXCEEDED from max_execution_time,
// MEMORY_LIMIT_EXCEEDED from max_memory_usage, and TOO_MANY_ROWS_OR_BYTES from
// max_result_bytes.
var readLimitCodes = map[int32]bool{
	159: true,
	241: true,
	396: true,
}

// classifyReadLimitError tags the limits [withReadLimits] sets so they reach
// the caller as rejections, and returns everything else untouched.
//
// It gates on the exception rather than on what [WrapClickHouseError] returns,
// for two reasons. That helper reads an unmatched failure as a bad customer
// query, which is wrong here because these queries are written in this package.
// And it matches the word "timeout" anywhere in a message, so a transport
// failure like "read tcp ...: i/o timeout" would otherwise be served as a query
// limit. Both belong in the 5xx metrics instead, where an outage is visible.
func classifyReadLimitError(err error) error {
	if chErr, ok := errors.AsType[*ch.Exception](err); !ok || !readLimitCodes[chErr.Code] {
		return err
	}

	return WrapClickHouseError(err)
}
