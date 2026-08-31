-- Log drain delivery attempts: one row for every customer endpoint call.
--
-- svc/logdrain writes one row per Deliver attempt so the dashboard can chart
-- drain health and show recent deliveries. Rows capture endpoint response
-- details and delivery performance.
--
-- Time fields are Int64 unix milliseconds, matching the audit log source and
-- the other raw event tables. 30 day TTL: this is operational telemetry, not
-- an audit trail.

CREATE TABLE `default`.`logdrain_deliveries_raw_v1`
(
    `workspace_id` String CODEC(ZSTD(1)),
    `drain_id`     String CODEC(ZSTD(1)),
    `stream`       LowCardinality(String),

    -- Wall-clock completion time of the customer endpoint call.
    `time`         Int64 CODEC(Delta, ZSTD(1)),
    -- 'success' | 'error'.
    `outcome`      LowCardinality(String),
    `events`       Int64,
    -- Duration of the customer endpoint call in milliseconds.
    `webhook_duration_ms` Int64,
    -- The uncompressed request body size. Headers are not included.
    `request_body_bytes` Int64,
    -- Zero and empty when no HTTP response was received.
    `response_status` Int32,
    `response_body` String CODEC(ZSTD(1)),
    -- Empty on success.
    `error`        String CODEC(ZSTD(1))
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(time))
ORDER BY (workspace_id, drain_id, time)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 10000,
         ttl_only_drop_parts = 1;
