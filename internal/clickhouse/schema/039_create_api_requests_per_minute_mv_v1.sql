-- +goose up
CREATE MATERIALIZED VIEW metrics.api_requests_per_minute_mv_v1 TO metrics.api_requests_per_minute_v1 AS
SELECT
    workspace_id,
    path,
    response_status,
    host,
    method,
    count(*) as count,
    toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
FROM
    metrics.raw_api_requests_v1
GROUP BY
    workspace_id,
    path,
    response_status,
    host,
    method,
    time;

-- +goose down
DROP VIEW metrics.api_requests_per_minute_mv_v1;
