package clickhouse

import (
	"context"
	"maps"
	"strconv"

	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	apiRequestTime      logfilter.Column = "time"
	apiRequestPath      logfilter.Column = "path"
	apiRequestHost      logfilter.Column = "host"
	apiRequestMethod    logfilter.Column = "method"
	apiRequestRequestID logfilter.Column = "request_id"
	apiRequestStatus    logfilter.Column = "response_status"
)

// ApiRequestFilters are the filters of one listing, one operator set per field
type ApiRequestFilters struct {
	Time      logfilter.Int
	Path      logfilter.Text
	Host      logfilter.Exact
	Method    logfilter.Exact
	RequestID logfilter.Exact
	Status    logfilter.Int
}

// ApiRequestCursor is the position of the last row of a page. Time and
// RequestID together are unique, so rows that share a millisecond with the
// last row still come on the next page
type ApiRequestCursor struct {
	Time      int64
	RequestID string
}

// ListApiRequestsRequest selects one page of a workspace's API request logs.
// RetentionStart is the oldest time in unix milliseconds the workspace may
// read, whatever Filters ask for. After, when set, returns only the rows that
// come after that position in the listing order
type ListApiRequestsRequest struct {
	WorkspaceID    string
	RetentionStart int64
	After          *ApiRequestCursor
	Filters        ApiRequestFilters
	Limit          int
}

// ApiRequestsPage is one page of API request logs, newest first. HasMore is
// true when more rows match
type ApiRequestsPage struct {
	Requests []ApiRequestLog
	HasMore  bool
}

// ApiRequestLog is one row of default.api_requests_raw_v2 as a listing
// returns it
type ApiRequestLog struct {
	RequestID       string   `ch:"request_id"`
	Time            int64    `ch:"time"`
	Host            string   `ch:"host"`
	Method          string   `ch:"method"`
	Path            string   `ch:"path"`
	RequestHeaders  []string `ch:"request_headers"`
	RequestBody     string   `ch:"request_body"`
	ResponseStatus  int32    `ch:"response_status"`
	ResponseHeaders []string `ch:"response_headers"`
	ResponseBody    string   `ch:"response_body"`
	Error           string   `ch:"error"`
	ServiceLatency  int64    `ch:"service_latency"`
}

func (c *Client) ListApiRequests(ctx context.Context, req ListApiRequestsRequest) (ApiRequestsPage, error) {
	b := logfilter.New()
	b.Int(apiRequestTime, req.Filters.Time)
	b.Text(apiRequestPath, req.Filters.Path)
	b.Exact(apiRequestHost, req.Filters.Host)
	b.Exact(apiRequestMethod, req.Filters.Method)
	b.Exact(apiRequestRequestID, req.Filters.RequestID)
	b.Int(apiRequestStatus, req.Filters.Status)
	where, filterParams := b.Build()

	params := map[string]string{
		"workspace_id":    req.WorkspaceID,
		"retention_start": strconv.FormatInt(req.RetentionStart, 10),
		"limit_plus_one":  strconv.Itoa(req.Limit + 1),
	}
	maps.Copy(params, filterParams)

	// The plain time bound is what prunes on the sorting key. The tuple only
	// orders the rows inside the cursor's millisecond
	afterCursor := ""
	if req.After != nil {
		afterCursor = `AND time <= {cursor_time:Int64}
			AND (time, request_id) < ({cursor_time:Int64}, {cursor_request_id:String})`
		params["cursor_time"] = strconv.FormatInt(req.After.Time, 10)
		params["cursor_request_id"] = req.After.RequestID
	}

	rows, err := Select[ApiRequestLog](withReadLimits(ctx, AnalyticsResultBytesMax), c.conn, `
		SELECT request_id, time, host, method, path, request_headers, request_body,
			response_status, response_headers, response_body, error, service_latency
		FROM default.api_requests_raw_v2
		WHERE workspace_id = {workspace_id:String}
			AND time >= {retention_start:Int64}
			AND `+where+`
			`+afterCursor+`
		ORDER BY time DESC, request_id DESC
		LIMIT {limit_plus_one:UInt64}`, params)
	if err != nil {
		return ApiRequestsPage{}, fault.Wrap(classifyReadLimitError(err), fault.Internal("failed to list api requests"))
	}

	return ApiRequestsPage{
		Requests: rows[:min(len(rows), req.Limit)],
		HasMore:  len(rows) > req.Limit,
	}, nil
}
