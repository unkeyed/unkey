CREATE TABLE  api_requests_perday_v1 (
    time DateTime,
    workspace_id String,
    path String,
    response_status Int,
    host String,
    -- Upper case HTTP method
    -- Examples: "GET", "POST", "PUT", "DELETE"
    method LowCardinality(String),
    count Int64
) ENGINE = SummingMergeTree()
ORDER BY
    (
        workspace_id,
        time,
        host,
        path,
        response_status,
        method
    );


    CREATE MATERIALIZED VIEW  api_requests_perday_mv_v1 TO api_requests_perday_v1 AS
    SELECT
        workspace_id,
        path,
        response_status,
        host,
        method,
        count(*) as count,
        toStartOfDay(fromUnixTimestamp64Milli(time)) AS time
    FROM
        raw_api_requests_v1
    GROUP BY
        workspace_id,
        path,
        response_status,
        host,
        method,
        time;
