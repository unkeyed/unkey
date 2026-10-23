package clickhouse

import (
	"context"
	"strconv"

	"github.com/unkeyed/unkey/pkg/clickhouse/logfilter"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	apiRequestTime      logfilter.Column = "time"
	apiRequestPath      logfilter.Column = "path"
	apiRequestMethod    logfilter.Column = "method"
	apiRequestRequestID logfilter.Column = "request_id"
	apiRequestStatus    logfilter.Column = "response_status"
)

// ApiRequestFilters are the filters of one listing, one operator set per field
type ApiRequestFilters struct {
	Time      logfilter.Int
	Path      logfilter.Text
	Method    logfilter.Exact
	RequestID logfilter.Exact
	Status    logfilter.Int
}

// ApiRequestCursor is the position of the first row of a page. Time and
// RequestID together are unique, so rows that share a millisecond with the
// cursor row still come on the page
type ApiRequestCursor struct {
	Time      int64
	RequestID string
}

// ListApiRequestsRequest selects one page of a workspace's API request logs.
// RetentionStart is the oldest time in unix milliseconds the workspace may
// read, whatever Filters ask for. Cursor, when set, returns the rows from it
// on in the listing order. Limit is the most rows to return
type ListApiRequestsRequest struct {
	WorkspaceID    string
	RetentionStart int64
	Cursor         *ApiRequestCursor
	Filters        ApiRequestFilters
	Limit          int
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
	ServiceLatency  int64    `ch:"service_latency"`
}

func (c *Client) ListApiRequests(ctx context.Context, req ListApiRequestsRequest) ([]ApiRequestLog, error) {
	b := logfilter.New()
	b.Int(apiRequestTime, req.Filters.Time)
	b.Text(apiRequestPath, req.Filters.Path)
	b.Exact(apiRequestMethod, req.Filters.Method)
	b.Exact(apiRequestRequestID, req.Filters.RequestID)
	b.Int(apiRequestStatus, req.Filters.Status)
	where, params := b.Build()
	params["workspace_id"] = req.WorkspaceID
	params["retention_start"] = strconv.FormatInt(req.RetentionStart, 10)
	params["row_limit"] = strconv.Itoa(req.Limit)

	cursorCondition := ""
	if req.Cursor != nil {
		cursorCondition = `AND time <= {cursor_time:Int64}
			AND (time, request_id) <= ({cursor_time:Int64}, {cursor_request_id:String})`
		params["cursor_time"] = strconv.FormatInt(req.Cursor.Time, 10)
		params["cursor_request_id"] = req.Cursor.RequestID
	}

	rows, err := Select[ApiRequestLog](withReadLimits(ctx), c.conn, `
		SELECT request_id, time, host, method, path, request_headers, request_body,
			response_status, response_headers, response_body, service_latency
		FROM default.api_requests_raw_v2
		WHERE workspace_id = {workspace_id:String}
			AND time >= {retention_start:Int64}
			AND `+where+`
			`+cursorCondition+`
		ORDER BY time DESC, request_id DESC
		LIMIT {row_limit:UInt64}`, params)
	if err != nil {
		return nil, fault.Wrap(classifyReadLimitError(err), fault.Internal("failed to list api requests"))
	}

	return rows, nil
}
