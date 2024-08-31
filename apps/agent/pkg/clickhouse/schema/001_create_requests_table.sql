-- +goose up
CREATE TABLE default.api_requests__v1(
    request_id String,
    -- unix milli
    time Int64,
    host String,
    method LowCardinality(String),
    path String,
    request Nested (
        -- "Key: Value" pairs
        headers Array(String),
        -- encoded as Content-Type, usually json
        body String
    ),

    response Nested(
        status Int,
        -- "Key: Value" pairs
        headers Array(String),
        -- encoded as Content-Type, usually json
        body String
    )

)
ENGINE = MergeTree()
PRIMARY KEY (request_id);
