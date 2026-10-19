package clickhouse

import (
	"context"
	"maps"
	"strconv"

	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	minuteMillis = 60 * 1000
	hourMillis   = 60 * minuteMillis
	dayMillis    = 24 * hourMillis
)

// apiRequestsInterval is one bucket width of the timeseries and the aggregate
// table that holds it. tableRetentionMs is the table's TTL in
// pkg/clickhouse/schema: a window that starts earlier would read deleted rows
type apiRequestsInterval struct {
	windowMaxMs      int64 // from the first to the last millisecond of the window
	bucketMs         int64
	table            string
	tableRetentionMs int64
}

// apiRequestsIntervals gives a window of an hour or more 24 to 121 buckets.
// The last one must cover every window the timeseries accepts
var apiRequestsIntervals = []apiRequestsInterval{
	{windowMaxMs: 2 * hourMillis, bucketMs: minuteMillis, table: "default.api_requests_per_minute_v2", tableRetentionMs: 7 * dayMillis},
	{windowMaxMs: 6 * hourMillis, bucketMs: 5 * minuteMillis, table: "default.api_requests_per_minute_v2", tableRetentionMs: 7 * dayMillis},
	{windowMaxMs: 12 * hourMillis, bucketMs: 15 * minuteMillis, table: "default.api_requests_per_minute_v2", tableRetentionMs: 7 * dayMillis},
	{windowMaxMs: dayMillis, bucketMs: 30 * minuteMillis, table: "default.api_requests_per_minute_v2", tableRetentionMs: 7 * dayMillis},
	{windowMaxMs: 3 * dayMillis, bucketMs: hourMillis, table: "default.api_requests_per_hour_v2", tableRetentionMs: 30 * dayMillis},
	{windowMaxMs: 7 * dayMillis, bucketMs: 2 * hourMillis, table: "default.api_requests_per_hour_v2", tableRetentionMs: 30 * dayMillis},
	{windowMaxMs: 30 * dayMillis, bucketMs: 6 * hourMillis, table: "default.api_requests_per_hour_v2", tableRetentionMs: 30 * dayMillis},
	{windowMaxMs: apiRequestsWindowMaxMs, bucketMs: dayMillis, table: "default.api_requests_per_day_v2", tableRetentionMs: apiRequestsWindowMaxMs},
}

// apiRequestsWindowMaxMs is the TTL of api_requests_per_day_v2, the oldest
// aggregate data there is
const apiRequestsWindowMaxMs = 100 * dayMillis

// ApiRequestsTimeseriesFilters are the [ApiRequestFilters] the aggregate
// tables can answer. They have no request id column
type ApiRequestsTimeseriesFilters struct {
	Time   logfilter.Int
	Path   logfilter.Text
	Host   logfilter.Exact
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

// ApiRequestsTimeseries is one point per bucket of width BucketMs, oldest
// first, with no gaps. The first bucket starts at or before the window start
// and counts its whole width
type ApiRequestsTimeseries struct {
	BucketMs int64
	Points   []ApiRequestsTimeseriesPoint
}

// ApiRequestsTimeseriesPoint counts the requests of the bucket that starts at
// Time. Total includes the 1xx and 3xx responses, which have no field of
// their own
type ApiRequestsTimeseriesPoint struct {
	Time        int64 `ch:"bucket_time"`
	Success     int64 `ch:"success"`
	ClientError int64 `ch:"client_error"`
	ServerError int64 `ch:"server_error"`
	Total       int64 `ch:"total"`
}

func (c *Client) GetApiRequestsTimeseries(ctx context.Context, req ApiRequestsTimeseriesRequest) (ApiRequestsTimeseries, error) {
	t := req.Filters.Time

	end := req.Now
	if t.Lt != nil {
		end = min(end, *t.Lt)
	}
	if t.Lte != nil && *t.Lte < end {
		end = *t.Lte + 1
	}

	start := req.RetentionStart
	if t.Gte != nil || t.Gt != nil {
		start = 0
		if t.Gte != nil {
			start = *t.Gte
		}
		if t.Gt != nil {
			if *t.Gt >= end {
				return ApiRequestsTimeseries{BucketMs: apiRequestsIntervals[0].bucketMs, Points: []ApiRequestsTimeseriesPoint{}}, nil
			}
			start = max(start, *t.Gt+1)
		}
	}
	start = max(start, req.Now-apiRequestsWindowMaxMs)

	if start >= end {
		return ApiRequestsTimeseries{BucketMs: apiRequestsIntervals[0].bucketMs, Points: []ApiRequestsTimeseriesPoint{}}, nil
	}

	iv := apiRequestsIntervals[len(apiRequestsIntervals)-1]
	for _, candidate := range apiRequestsIntervals {
		if end-1-start <= candidate.windowMaxMs && req.Now-start <= candidate.tableRetentionMs {
			iv = candidate
			break
		}
	}

	firstBucket := start - start%iv.bucketMs
	lastBucket := (end - 1) - (end-1)%iv.bucketMs

	b := logfilter.New()
	b.Text(apiRequestPath, req.Filters.Path)
	b.Exact(apiRequestHost, req.Filters.Host)
	b.Exact(apiRequestMethod, req.Filters.Method)
	b.Int(apiRequestStatus, req.Filters.Status)
	where, filterParams := b.Build()

	params := map[string]string{
		"workspace_id":   req.WorkspaceID,
		"rows_start":     strconv.FormatInt(max(firstBucket, req.RetentionStart), 10),
		"rows_end":       strconv.FormatInt(end, 10),
		"bucket_seconds": strconv.FormatInt(iv.bucketMs/1000, 10),
		"bucket_ms":      strconv.FormatInt(iv.bucketMs, 10),
		"fill_from":      strconv.FormatInt(firstBucket, 10),
		"fill_to":        strconv.FormatInt(lastBucket+iv.bucketMs, 10),
	}
	maps.Copy(params, filterParams)

	// The day table stores a Date, so time converts as UTC midnight, the same
	// edge the epoch-aligned buckets of every other interval fall on
	points, err := Select[ApiRequestsTimeseriesPoint](withReadLimits(ctx, AnalyticsResultBytesMax), c.conn, `
		SELECT
			toInt64(intDiv(toUnixTimestamp(toDateTime(time, 'UTC')), {bucket_seconds:Int64}) * {bucket_ms:Int64}) AS bucket_time,
			sumIf(count, response_status >= 200 AND response_status < 300) AS success,
			sumIf(count, response_status >= 400 AND response_status < 500) AS client_error,
			sumIf(count, response_status >= 500) AS server_error,
			sum(count) AS total
		FROM `+iv.table+`
		WHERE workspace_id = {workspace_id:String}
			AND time >= fromUnixTimestamp64Milli({rows_start:Int64})
			AND time < fromUnixTimestamp64Milli({rows_end:Int64})
			AND `+where+`
		GROUP BY bucket_time
		ORDER BY bucket_time ASC
		WITH FILL FROM {fill_from:Int64} TO {fill_to:Int64} STEP {bucket_ms:Int64}`, params)
	if err != nil {
		return ApiRequestsTimeseries{}, fault.Wrap(classifyReadLimitError(err), fault.Internal("failed to read the api requests timeseries"))
	}

	return ApiRequestsTimeseries{
		BucketMs: iv.bucketMs,
		Points:   points,
	}, nil
}
