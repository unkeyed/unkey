-- +goose up
CREATE TABLE ratelimits.raw_ratelimits_v1(
    request_id    String,
    -- unix milli
    time          Int64,
    workspace_id  String,
    namespace_id  String,
    identifier    String,
    passed        Bool

)
ENGINE = MergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;



-- +goose down
DROP TABLE ratelimits.raw_ratelimits_v1;
