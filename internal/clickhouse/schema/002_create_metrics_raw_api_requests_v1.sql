-- +goose up
CREATE TABLE metrics.raw_api_requests_v1(
    request_id String,
    -- unix milli
    time Int64,

    workspace_id String,

    host String,

    -- Upper case HTTP method
    -- Examples: "GET", "POST", "PUT", "DELETE"
    method LowCardinality(String),
    path String,
    -- "Key: Value" pairs
    request_headers Array(String),
    request_body String,

    response_status Int,
    -- "Key: Value" pairs
    response_headers Array(String),
    response_body String,
    -- internal err.Error() string, empty if no error
    error String,

    -- milliseconds
    service_latency Int64,

    user_agent String,
    ip_address String,
    country String,
    city String,
    colo String,
    continent String,
  

)
ENGINE = MergeTree()
ORDER BY (workspace_id, time, request_id)
;

-- +goose down
DROP TABLE metrics.raw_api_requests_v1;
