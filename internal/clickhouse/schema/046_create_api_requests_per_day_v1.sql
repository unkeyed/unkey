-- +goose up
CREATE TABLE metrics.api_requests_per_day_v1 (
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

-- +goose down
DROP TABLE metrics.api_requests_per_day_v1;
