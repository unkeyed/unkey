package clickhouse

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/fault"
)

// VerificationTimeseriesRequest scopes a verification timeseries to a single
// portal end user within an explicit set of keyspaces. WorkspaceID, ExternalID,
// and KeyspaceIDs are required; KeyID optionally narrows to one key. StartTime
// and EndTime bound the window in unix milliseconds (StartTime inclusive,
// EndTime exclusive).
type VerificationTimeseriesRequest struct {
	WorkspaceID string
	ExternalID  string
	KeyspaceIDs []string
	KeyID       string
	StartTime   int64
	EndTime     int64
}

// VerificationTimeseriesDataPoint is one time bucket of verification counts
// broken out by outcome. Time is the bucket start in unix milliseconds. The
// outcome fields mirror the dashboard's verification timeseries shape so portal
// charts can reuse the same components. The ch tags map onto the query's column
// aliases for [Select].
type VerificationTimeseriesDataPoint struct {
	Time                    int64 `ch:"x"`
	Total                   int64 `ch:"total"`
	Valid                   int64 `ch:"valid"`
	RateLimited             int64 `ch:"rate_limited"`
	InsufficientPermissions int64 `ch:"insufficient_permissions"`
	Forbidden               int64 `ch:"forbidden"`
	Disabled                int64 `ch:"disabled"`
	Expired                 int64 `ch:"expired"`
	UsageExceeded           int64 `ch:"usage_exceeded"`
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
// one end user within the requested keyspaces, optionally narrowed to a single
// key. Bucket granularity is chosen from the window size. Empty buckets are
// returned with zero counts so callers get a contiguous series.
//
// The query runs on the shared ClickHouse connection (not a per-workspace user)
// and filters on key_space_id and external_id, which are denormalized onto each
// event at write time. This is the portal-scoped read: the workspace, keyspaces,
// and identity are pinned by the caller, so no query DSL or per-workspace
// connection is involved.
func (c *Client) GetVerificationsByExternalID(ctx context.Context, req VerificationTimeseriesRequest) ([]VerificationTimeseriesDataPoint, error) {
	iv := selectVerificationInterval(req.EndTime - req.StartTime)

	// iv.unit, iv.table and iv.stepMs come from selectVerificationInterval — a
	// fixed switch over the window size, never caller input — so they are safe to
	// interpolate. Every caller-supplied value (workspace, identity, key, window
	// bounds) goes through a typed query parameter instead. SUM results are cast
	// to Int64 so they scan into the int64 struct fields. An empty key_id means
	// "all keys": the OR short-circuits the filter rather than binding it.
	query := fmt.Sprintf(`
	SELECT
		toUnixTimestamp64Milli(CAST(toStartOfInterval(time, INTERVAL 1 %[1]s) AS DateTime64(3))) AS x,
		toInt64(SUM(count)) AS total,
		toInt64(SUM(IF(outcome = 'VALID', count, 0))) AS valid,
		toInt64(SUM(IF(outcome = 'RATE_LIMITED', count, 0))) AS rate_limited,
		toInt64(SUM(IF(outcome = 'INSUFFICIENT_PERMISSIONS', count, 0))) AS insufficient_permissions,
		toInt64(SUM(IF(outcome = 'FORBIDDEN', count, 0))) AS forbidden,
		toInt64(SUM(IF(outcome = 'DISABLED', count, 0))) AS disabled,
		toInt64(SUM(IF(outcome = 'EXPIRED', count, 0))) AS expired,
		toInt64(SUM(IF(outcome = 'USAGE_EXCEEDED', count, 0))) AS usage_exceeded
	FROM %[2]s
	WHERE workspace_id = {workspace_id:String}
		AND external_id = {external_id:String}
		AND key_space_id IN {keyspace_ids:Array(String)}
		AND time >= fromUnixTimestamp64Milli({start:Int64})
		AND time < fromUnixTimestamp64Milli({end:Int64})
		AND ({key_id:String} = '' OR key_id = {key_id:String})
	GROUP BY x
	ORDER BY x ASC
	WITH FILL
		FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli({start:Int64}), INTERVAL 1 %[1]s) AS DateTime64(3)))
		TO toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli({end:Int64}), INTERVAL 1 %[1]s) AS DateTime64(3))) + %[3]d
		STEP %[3]d`,
		iv.unit, iv.table, iv.stepMs,
	)

	keyspaceIDs := array.Map(req.KeyspaceIDs, func(keyspaceID string) string {
		return fmt.Sprintf("'%s'", keyspaceID)
	})

	results, err := Select[VerificationTimeseriesDataPoint](ctx, c.conn, query, map[string]string{
		"workspace_id": req.WorkspaceID,
		"external_id":  req.ExternalID,
		"keyspace_ids": fmt.Sprintf("[%s]", strings.Join(keyspaceIDs, ",")),
		"key_id":       req.KeyID,
		"start":        strconv.FormatInt(req.StartTime, 10),
		"end":          strconv.FormatInt(req.EndTime, 10),
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to query verification timeseries"))
	}

	return results, nil
}
