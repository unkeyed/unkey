package clickhouse

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/fault"
)

// VerificationTimeseriesRequest scopes a verification timeseries to a single
// portal end user. WorkspaceID and ExternalID are required; KeyID optionally
// narrows to one key. StartTime and EndTime bound the window in unix
// milliseconds (StartTime inclusive, EndTime exclusive).
type VerificationTimeseriesRequest struct {
	WorkspaceID string
	ExternalID  string
	KeyID       string
	StartTime   int64
	EndTime     int64
}

// VerificationTimeseriesDataPoint is one time bucket of verification counts
// broken out by outcome. Time is the bucket start in unix milliseconds. The
// outcome fields mirror the dashboard's verification timeseries shape so portal
// charts can reuse the same components.
type VerificationTimeseriesDataPoint struct {
	Time                    int64
	Total                   int64
	Valid                   int64
	RateLimited             int64
	InsufficientPermissions int64
	Forbidden               int64
	Disabled                int64
	Expired                 int64
	UsageExceeded           int64
}

// verificationInterval describes the aggregated table and bucket width used for
// a given window size.
type verificationInterval struct {
	table  string
	unit   string // ClickHouse INTERVAL unit: minute, hour, or day
	stepMs int64  // bucket width in milliseconds, used for WITH FILL
}

// selectVerificationInterval picks the bucket granularity from the window
// duration, mirroring how the dashboard trades resolution for range: minute
// buckets for short windows, hour buckets for a few days, day buckets beyond.
func selectVerificationInterval(windowMs int64) verificationInterval {
	switch {
	case windowMs <= 3*60*60*1000: // <= 3 hours
		return verificationInterval{"default.key_verifications_per_minute_v3", "minute", 60 * 1000}
	case windowMs <= 4*24*60*60*1000: // <= 4 days
		return verificationInterval{"default.key_verifications_per_hour_v3", "hour", 60 * 60 * 1000}
	default:
		return verificationInterval{"default.key_verifications_per_day_v3", "day", 24 * 60 * 60 * 1000}
	}
}

// GetVerificationsByExternalID returns a zero-filled verification timeseries for
// one end user (workspace_id + external_id), optionally narrowed to a single
// key. Bucket granularity is chosen from the window size. Empty buckets are
// returned with zero counts so callers get a contiguous series.
//
// The query runs on the shared ClickHouse connection (not a per-workspace user)
// and filters on external_id, which is denormalized onto each event at write
// time. This is the portal-scoped read: the workspace and identity are pinned by
// the caller, so no query DSL or per-workspace connection is involved.
func (c *Client) GetVerificationsByExternalID(ctx context.Context, req VerificationTimeseriesRequest) ([]VerificationTimeseriesDataPoint, error) {
	iv := selectVerificationInterval(req.EndTime - req.StartTime)

	keyFilter := ""
	args := []any{req.WorkspaceID, req.ExternalID, req.StartTime, req.EndTime}
	if req.KeyID != "" {
		keyFilter = "AND key_id = ?"
		args = append(args, req.KeyID)
	}
	// WITH FILL bounds reuse the window; appended after the optional key filter
	// so positional args stay aligned with the placeholders in the query.
	args = append(args, req.StartTime, req.EndTime)

	query := fmt.Sprintf(`
	SELECT
		toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL 1 %[1]s) AS DateTime64(3))) AS x,
		SUM(count) AS total,
		SUM(IF(outcome = 'VALID', count, 0)) AS valid,
		SUM(IF(outcome = 'RATE_LIMITED', count, 0)) AS rate_limited,
		SUM(IF(outcome = 'INSUFFICIENT_PERMISSIONS', count, 0)) AS insufficient_permissions,
		SUM(IF(outcome = 'FORBIDDEN', count, 0)) AS forbidden,
		SUM(IF(outcome = 'DISABLED', count, 0)) AS disabled,
		SUM(IF(outcome = 'EXPIRED', count, 0)) AS expired,
		SUM(IF(outcome = 'USAGE_EXCEEDED', count, 0)) AS usage_exceeded
	FROM %[2]s
	WHERE workspace_id = ?
		AND external_id = ?
		AND time >= fromUnixTimestamp64Milli(?)
		AND time < fromUnixTimestamp64Milli(?)
		%[3]s
	GROUP BY x
	ORDER BY x ASC
	WITH FILL
		FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli(?), INTERVAL 1 %[1]s) AS DateTime64(3)))
		TO toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli(?), INTERVAL 1 %[1]s) AS DateTime64(3))) + %[4]d
		STEP %[4]d`,
		iv.unit, iv.table, keyFilter, iv.stepMs,
	)

	rows, err := c.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query verification timeseries"))
	}
	defer func() { _ = rows.Close() }()

	var results []VerificationTimeseriesDataPoint
	for rows.Next() {
		var r VerificationTimeseriesDataPoint
		if err := rows.Scan(
			&r.Time, &r.Total, &r.Valid, &r.RateLimited, &r.InsufficientPermissions,
			&r.Forbidden, &r.Disabled, &r.Expired, &r.UsageExceeded,
		); err != nil {
			return nil, fault.Wrap(err, fault.Internal("failed to scan verification timeseries row"))
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fault.Wrap(err, fault.Internal("error iterating verification timeseries rows"))
	}

	return results, nil
}
