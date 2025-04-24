CREATE TABLE IF NOT EXISTS ratelimits.raw_ratelimits_v1(
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

ALTER TABLE ratelimits.raw_ratelimits_v1 ADD INDEX IF NOT EXISTS idx_workspace_time (workspace_id, time) TYPE minmax GRANULARITY 1;

ALTER TABLE ratelimits.raw_ratelimits_v1
ADD INDEX IF NOT EXISTS idx_request_id (request_id) TYPE minmax GRANULARITY 1;
