CREATE TABLE IF NOT EXISTS default.outpost_requests_raw_v1
(
    `request_id` String,
    `time` Int64 CODEC(Delta, LZ4),
    `outpost_id` String,
    `sentinel_request_id` String,
    `workspace_id` String,
    `deployment_id` String,
    `region` LowCardinality(String),
    `destination_host` String,
    `method` LowCardinality(String),
    `path` String,
    `response_status` Int32,
    `latency_ms` Int64,
    `request_bytes` Int64,
    `response_bytes` Int64,
    `error` String,
    INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
    INDEX idx_deployment_id (deployment_id) TYPE bloom_filter GRANULARITY 1,
    INDEX idx_destination_host (destination_host) TYPE bloom_filter GRANULARITY 1,
    INDEX idx_sentinel_request_id (sentinel_request_id) TYPE bloom_filter GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (workspace_id, deployment_id, time)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(30)
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
