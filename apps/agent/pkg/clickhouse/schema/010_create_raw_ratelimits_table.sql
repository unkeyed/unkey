-- +goose up
CREATE TABLE default.raw_ratelimits_v1(
    request_id String,
    -- unix milli
    time Int64,

    workspace_id String,

    namespace_id String,

    identifier String,
    success Int8

)
ENGINE = MergeTree()
ORDER BY (workspace_id, namespace_id, time, request_id)
;
