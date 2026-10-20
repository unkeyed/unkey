package clickhouse

import (
	"context"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
	"github.com/unkeyed/unkey/pkg/fault"
)

var (
	minuteMs = time.Minute.Milliseconds()
	hourMs   = time.Hour.Milliseconds()
	dayMs    = 24 * hourMs
)

// apiRequestsTable is an aggregate table of the API requests. retentionMs is
// its TTL in pkg/clickhouse/schema: a window that starts earlier would read
// deleted rows
type apiRequestsTable struct {
	name        string
	retentionMs int64
}

var (
	apiRequestsPerMinute = apiRequestsTable{name: "default.api_requests_per_minute_v2", retentionMs: 7 * dayMs}
	apiRequestsPerHour   = apiRequestsTable{name: "default.api_requests_per_hour_v2", retentionMs: 30 * dayMs}
	apiRequestsPerDay    = apiRequestsTable{name: "default.api_requests_per_day_v2", retentionMs: 100 * dayMs}
)

// apiRequestsResolution is one bucket width of the timeseries and the table
// that holds it
type apiRequestsResolution struct {
	windowMsMax int64 // from the first to the last millisecond of the window
	bucketMs    int64
	table       apiRequestsTable
}

// Each resolution gives a window of an hour or more between 24 and 121
// buckets. The last one covers the longest window the timeseries accepts,
// which is all the data the day table keeps
var apiRequestsResolutions = []apiRequestsResolution{
	{windowMsMax: 2 * hourMs, bucketMs: minuteMs, table: apiRequestsPerMinute},
	{windowMsMax: 6 * hourMs, bucketMs: 5 * minuteMs, table: apiRequestsPerMinute},
	{windowMsMax: 12 * hourMs, bucketMs: 15 * minuteMs, table: apiRequestsPerMinute},
	{windowMsMax: dayMs, bucketMs: 30 * minuteMs, table: apiRequestsPerMinute},
	{windowMsMax: 3 * dayMs, bucketMs: hourMs, table: apiRequestsPerHour},
	{windowMsMax: 7 * dayMs, bucketMs: 2 * hourMs, table: apiRequestsPerHour},
	{windowMsMax: 30 * dayMs, bucketMs: 6 * hourMs, table: apiRequestsPerHour},
	{windowMsMax: apiRequestsPerDay.retentionMs, bucketMs: dayMs, table: apiRequestsPerDay},
}

// ApiRequestsTimeseriesFilters are the [ApiRequestFilters] the aggregate
// tables can answer. They have no request id column
type ApiRequestsTimeseriesFilters struct {
	Time   logfilter.Int
	Path   logfilter.Text
	Method logfilter.Exact
	Status logfilter.Int
}

// ApiRequestsTimeseriesRequest counts a workspace's API requests per bucket.
// Filters.Time sets the window. A missing lower bound is RetentionStart and a
// missing upper bound is Now. The window never reaches past Now or further
// back than the aggregate tables keep data, and rows older than
// RetentionStart are never counted, whatever the window. Now is unix
// milliseconds
type ApiRequestsTimeseriesRequest struct {
	WorkspaceID    string
	Now            int64
	RetentionStart int64
	Filters        ApiRequestsTimeseriesFilters
}

// ApiRequestsTimeseriesPoint counts the requests of the bucket that starts at
// Time. A timeseries has one point per bucket, oldest first, with no gaps, so
// the bucket width is the distance between two points. The first bucket starts
// at or before the window start and counts its whole width, except rows older
// than RetentionStart. Total includes the 1xx and 3xx responses, which have no field of
// their own
type ApiRequestsTimeseriesPoint struct {
	Time        int64 `ch:"bucket_time"`
	Success     int64 `ch:"success"`
	ClientError int64 `ch:"client_error"`
	ServerError int64 `ch:"server_error"`
	Total       int64 `ch:"total"`
}

func (c *Client) GetApiRequestsTimeseries(ctx context.Context, req ApiRequestsTimeseriesRequest) ([]ApiRequestsTimeseriesPoint, error) {
	timeFilter := req.Filters.Time

	// min before +1 keeps a bound near MaxInt64 from wrapping around
	windowEnd := req.Now
	if timeFilter.Lt != nil {
		windowEnd = min(windowEnd, *timeFilter.Lt)
	}
	if timeFilter.Lte != nil {
		windowEnd = min(windowEnd-1, *timeFilter.Lte) + 1
	}

	// A lower bound the caller sets starts the buckets there, even before
	// RetentionStart, so a fixed window keeps its buckets. Rows before
	// RetentionStart still read as zero
	windowStart := req.RetentionStart
	if timeFilter.Gte != nil || timeFilter.Gt != nil {
		windowStart = 0
	}
	if timeFilter.Gte != nil {
		windowStart = max(windowStart, *timeFilter.Gte)
	}
	if timeFilter.Gt != nil {
		windowStart = max(windowStart, min(*timeFilter.Gt, windowEnd)+1)
	}
	windowStart = max(windowStart, req.Now-apiRequestsPerDay.retentionMs)

	if windowStart >= windowEnd {
		return []ApiRequestsTimeseriesPoint{}, nil
	}

	windowLast := windowEnd - 1

	resolution := apiRequestsResolutions[len(apiRequestsResolutions)-1]
	for _, candidate := range apiRequestsResolutions {
		if windowLast-windowStart <= candidate.windowMsMax && req.Now-windowStart <= candidate.table.retentionMs {
			resolution = candidate
			break
		}
	}

	firstBucketStart := windowStart - windowStart%resolution.bucketMs
	lastBucketStart := windowLast - windowLast%resolution.bucketMs

	b := logfilter.New()
	b.Text(apiRequestPath, req.Filters.Path)
	b.Exact(apiRequestMethod, req.Filters.Method)
	b.Int(apiRequestStatus, req.Filters.Status)
	where, params := b.Build()
	params["workspace_id"] = req.WorkspaceID
	params["rows_start"] = strconv.FormatInt(max(firstBucketStart, req.RetentionStart), 10)
	params["rows_end"] = strconv.FormatInt(windowEnd, 10)
	params["bucket_seconds"] = strconv.FormatInt(resolution.bucketMs/time.Second.Milliseconds(), 10)
	params["bucket_ms"] = strconv.FormatInt(resolution.bucketMs, 10)
	params["fill_from"] = strconv.FormatInt(firstBucketStart, 10)
	params["fill_to"] = strconv.FormatInt(lastBucketStart+resolution.bucketMs, 10)

	// The day table's time is a Date, and toDateTime(time, 'UTC') makes it UTC
	// midnight, where an epoch-aligned day bucket starts
	points, err := Select[ApiRequestsTimeseriesPoint](withReadLimits(ctx), c.conn, `
		SELECT
			toInt64(intDiv(toUnixTimestamp(toDateTime(time, 'UTC')), {bucket_seconds:Int64}) * {bucket_ms:Int64}) AS bucket_time,
			sumIf(count, response_status >= 200 AND response_status < 300) AS success,
			sumIf(count, response_status >= 400 AND response_status < 500) AS client_error,
			sumIf(count, response_status >= 500) AS server_error,
			sum(count) AS total
		FROM `+resolution.table.name+`
		WHERE workspace_id = {workspace_id:String}
			AND time >= fromUnixTimestamp64Milli({rows_start:Int64})
			AND time < fromUnixTimestamp64Milli({rows_end:Int64})
			AND `+where+`
		GROUP BY bucket_time
		ORDER BY bucket_time ASC
		WITH FILL FROM {fill_from:Int64} TO {fill_to:Int64} STEP {bucket_ms:Int64}`, params)
	if err != nil {
		return nil, fault.Wrap(classifyReadLimitError(err), fault.Internal("failed to read the api requests timeseries"))
	}

	return points, nil
}
