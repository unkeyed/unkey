-- +goose up
CREATE TABLE default.api_requests__v1(
    request_id String,
    -- unix milli
    time Int64,
    host String,
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
    error String

)
ENGINE = MergeTree()
PRIMARY KEY (request_id);
