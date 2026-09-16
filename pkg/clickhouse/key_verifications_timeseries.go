package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ErrTooManyVerificationKeys is returned by
// [Client.GetVerificationsByExternalIDPerKey] when the end user has more keys
// with traffic in the window than the caller allowed. Callers turn this into a
// user-facing rejection; a truncated breakout is indistinguishable from those
// keys having no traffic, so it is never returned.
var ErrTooManyVerificationKeys = errors.New("too many keys for a per-key verification breakout")

// VerificationTimeseriesRequest scopes a verification timeseries to a single
// portal end user. WorkspaceID, ExternalID and KeySpaceIDs are required; KeyID
// optionally narrows to one key. StartTime and EndTime bound the window in unix
// milliseconds (StartTime inclusive, EndTime exclusive).
type VerificationTimeseriesRequest struct {
	WorkspaceID string
	ExternalID  string
	KeySpaceIDs []string
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

const (
	// portalQueryExecutionTimeMax bounds how long one portal read may occupy the
	// shared connection, in seconds.
	portalQueryExecutionTimeMax = 10
	// portalQueryMemoryMax bounds one portal read's server-side memory.
	portalQueryMemoryMax = 1 << 30
)

// withPortalQueryLimits bounds a single portal read. The operator analytics
// routes get the equivalent from a per-workspace ClickHouse settings profile,
// which this path has no user to hang off: it runs on the connection every
// workspace shares, so the bounds ride on the query instead.
//
// max_result_bytes stops an oversized read server-side, before the rows are
// shipped. The handler's own ceiling still applies to the encoded response,
// which covers both reads plus their JSON.
func withPortalQueryLimits(ctx context.Context) context.Context {
	return ch.Context(ctx, ch.WithSettings(ch.Settings{
		"max_execution_time":   portalQueryExecutionTimeMax,
		"max_memory_usage":     portalQueryMemoryMax,
		"max_result_bytes":     AnalyticsResultBytesMax,
		"result_overflow_mode": "throw",
	}))
}

// portalQueryLimitCodes are the ClickHouse exceptions the three settings in
// [withPortalQueryLimits] can raise: TIMEOUT_EXCEEDED from max_execution_time,
// MEMORY_LIMIT_EXCEEDED from max_memory_usage, and TOO_MANY_ROWS_OR_BYTES from
// max_result_bytes.
var portalQueryLimitCodes = map[int32]bool{
	159: true,
	241: true,
	396: true,
}

// classifyPortalQueryError tags the limits [withPortalQueryLimits] sets so they
// reach the caller as rejections, and returns everything else untouched.
//
// It gates on the exception rather than on what [WrapClickHouseError] returns,
// for two reasons. That helper reads an unmatched failure as a bad customer
// query, which is wrong here because these queries are written in this package.
// And it matches the word "timeout" anywhere in a message, so a transport
// failure like "read tcp ...: i/o timeout" would otherwise be served as a query
// limit. Both belong in the 5xx metrics instead, where an outage is visible.
func classifyPortalQueryError(err error) error {
	var chErr *ch.Exception
	if !errors.As(err, &chErr) || !portalQueryLimitCodes[chErr.Code] {
		return err
	}

	return WrapClickHouseError(err)
}

// verificationScopePredicates is the WHERE body shared by the account-wide and
// per-key reads. Keeping one copy is what stops the identity and keyspace
// scoping from drifting between them.
const verificationScopePredicates = `workspace_id = {workspace_id:String}
		AND external_id = {external_id:String}
		AND key_space_id IN {key_space_ids:Array(String)}
		AND time >= fromUnixTimestamp64Milli({start:Int64})
		AND time < fromUnixTimestamp64Milli({end:Int64})
		AND ({key_id:String} = '' OR key_id = {key_id:String})`

// GetVerificationsByExternalID returns a zero-filled verification timeseries for
// one end user (workspace_id + external_id), optionally narrowed to a single
// key. Bucket granularity is chosen from the window size. Empty buckets are
// returned with zero counts so callers get a contiguous series.
//
// The query runs on the shared ClickHouse connection (not a per-workspace user)
// and filters on external_id, which is denormalized onto each event at write
// time. This is the portal-scoped read: the workspace, identity and keyspaces
// are pinned by the caller, so no query DSL or per-workspace connection is
// involved.
func (c *Client) GetVerificationsByExternalID(ctx context.Context, req VerificationTimeseriesRequest) ([]VerificationTimeseriesDataPoint, error) {
	if err := req.requireKeySpaceScope(); err != nil {
		return nil, err
	}

	iv := selectVerificationInterval(req.EndTime - req.StartTime)

	// iv.unit, iv.table and iv.stepMs come from selectVerificationInterval — a
	// fixed switch over the window size, never caller input — so they are safe to
	// interpolate. Every caller-supplied value (workspace, identity, keyspaces,
	// key, window bounds) goes through a typed named parameter instead. SUM
	// results are cast to Int64 so they scan into the int64 struct fields. An
	// empty key_id means "all keys": the OR short-circuits the filter rather
	// than binding it.
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
	WHERE %[4]s
	GROUP BY x
	ORDER BY x ASC
	WITH FILL
		FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli({start:Int64}), INTERVAL 1 %[1]s) AS DateTime64(3)))
		TO toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli({end:Int64}), INTERVAL 1 %[1]s) AS DateTime64(3))) + %[3]d
		STEP %[3]d`,
		iv.unit, iv.table, iv.stepMs, verificationScopePredicates,
	)

	results, err := Select[VerificationTimeseriesDataPoint](withPortalQueryLimits(ctx), c.conn, query, verificationScopeParams(req))
	if err != nil {
		return nil, fault.Wrap(classifyPortalQueryError(err), fault.Internal("failed to query verification timeseries"))
	}

	return results, nil
}

// requireKeySpaceScope rejects a read that lost its keyspace scope. Callers
// derive the list from the portal session, so an empty one is a broken
// invariant, and answering it would widen the read past the portal.
func (req VerificationTimeseriesRequest) requireKeySpaceScope() error {
	if len(req.KeySpaceIDs) > 0 {
		return nil
	}

	return fault.New("missing keyspace scope",
		fault.Code(codes.App.Internal.UnexpectedError.URN()),
		fault.Internal("verification timeseries requested with no key spaces"),
		fault.Public("An internal error occurred."),
	)
}

// verificationScopeParams binds the values [verificationScopePredicates] reads.
func verificationScopeParams(req VerificationTimeseriesRequest) map[string]string {
	return map[string]string{
		"workspace_id":  req.WorkspaceID,
		"external_id":   req.ExternalID,
		"key_space_ids": StringArrayParam(req.KeySpaceIDs),
		"key_id":        req.KeyID,
		"start":         strconv.FormatInt(req.StartTime, 10),
		"end":           strconv.FormatInt(req.EndTime, 10),
	}
}

// VerificationTimeseriesPerKeyRequest is a [VerificationTimeseriesRequest] plus
// the largest number of distinct keys the caller is willing to receive.
type VerificationTimeseriesPerKeyRequest struct {
	VerificationTimeseriesRequest
	MaxKeys int
}

// VerificationTimeseriesPerKey is one key's verification series, zero-filled
// across the requested window and ordered by time ascending.
type VerificationTimeseriesPerKey struct {
	KeyID string
	Data  []VerificationTimeseriesDataPoint
}

// verificationTimeseriesPerKeyRow is one (key, bucket) row as ClickHouse
// returns it, before grouping into [VerificationTimeseriesPerKey].
type verificationTimeseriesPerKeyRow struct {
	KeyID string `ch:"key_id"`
	VerificationTimeseriesDataPoint
}

// GetVerificationsByExternalIDPerKey returns the same window as
// [Client.GetVerificationsByExternalID], broken out per key, under identical
// workspace, identity, keyspace and window scoping.
//
// Each series is zero-filled across the window, so a caller charting a subset
// of keys gets contiguous buckets without rebuilding them. That costs a full
// bucket run per key, which is why req.MaxKeys is set well below what the
// response size ceiling would otherwise allow.
//
// A key with no traffic at all in the window is still absent: there is no group
// for it to fill against.
//
// req.MaxKeys bounds how many distinct keys a session can pull over the shared
// connection. Exceeding it returns [ErrTooManyVerificationKeys] rather than a
// short array, which a caller could not tell apart from those keys being idle.
func (c *Client) GetVerificationsByExternalIDPerKey(ctx context.Context, req VerificationTimeseriesPerKeyRequest) ([]VerificationTimeseriesPerKey, error) {
	if err := req.requireKeySpaceScope(); err != nil {
		return nil, err
	}

	if req.MaxKeys <= 0 {
		return nil, fault.New("missing key cap",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("per-key verification timeseries requested with no key cap"),
			fault.Public("An internal error occurred."),
		)
	}

	iv := selectVerificationInterval(req.EndTime - req.StartTime)

	// The inner scan restricts the outer one to the first MaxKeys+1 keys, so an
	// over-cap request is detected without ever materializing every key's series.
	query := fmt.Sprintf(`
	SELECT
		key_id,
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
	WHERE %[3]s
		AND key_id IN (
			SELECT key_id
			FROM %[2]s
			WHERE %[3]s
			GROUP BY key_id
			ORDER BY key_id ASC
			LIMIT {max_keys_probe:UInt64}
		)
	GROUP BY key_id, x
	ORDER BY key_id ASC, x ASC
	WITH FILL
		FROM toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli({start:Int64}), INTERVAL 1 %[1]s) AS DateTime64(3)))
		TO toUnixTimestamp64Milli(CAST(toStartOfInterval(fromUnixTimestamp64Milli({end:Int64}), INTERVAL 1 %[1]s) AS DateTime64(3))) + %[4]d
		STEP %[4]d`,
		iv.unit, iv.table, verificationScopePredicates, iv.stepMs,
	)

	params := verificationScopeParams(req.VerificationTimeseriesRequest)
	params["max_keys_probe"] = strconv.Itoa(req.MaxKeys + 1)

	rows, err := Select[verificationTimeseriesPerKeyRow](withPortalQueryLimits(ctx), c.conn, query, params)
	if err != nil {
		return nil, fault.Wrap(classifyPortalQueryError(err), fault.Internal("failed to query per-key verification timeseries"))
	}

	series := make([]VerificationTimeseriesPerKey, 0)
	for _, row := range rows {
		// WITH FILL generates the bucket range even when a request matched no
		// rows at all, defaulting the grouping column. A real key id is never
		// empty, so those phantom rows are dropped rather than returned as a
		// key the caller does not own.
		if row.KeyID == "" {
			continue
		}

		if len(series) == 0 || series[len(series)-1].KeyID != row.KeyID {
			if len(series) == req.MaxKeys {
				return nil, ErrTooManyVerificationKeys
			}
			series = append(series, VerificationTimeseriesPerKey{
				KeyID: row.KeyID,
				Data:  nil,
			})
		}

		current := &series[len(series)-1]
		current.Data = append(current.Data, row.VerificationTimeseriesDataPoint)
	}

	return series, nil
}
