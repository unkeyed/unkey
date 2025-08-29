CREATE TABLE IF NOT EXISTS ratelimits.raw_ratelimits_v2(
    -- the request id for correlation with traces and logs
    request_id String,

    -- unix milli
    time Int64 CODEC(Delta, LZ4),

    workspace_id String,
    namespace_id String,
    identifier String,

    -- whether the ratelimit check passed or was blocked
    passed Bool,

    -- Latency in milliseconds for this ratelimit check
    latency Float64


)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(time))
ORDER BY (workspace_id, time, namespace_id, identifier)
TTL fromUnixTimestamp64Milli(time) + INTERVAL 100 DAY
SETTINGS non_replicated_deduplication_window = 10000
;

ALTER TABLE ratelimits.raw_ratelimits_v2
ADD INDEX IF NOT EXISTS idx_request_id (request_id) TYPE minmax GRANULARITY 1;
