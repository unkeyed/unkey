-- Runtime logs table for customer deployment logs
-- Stores stdout/stderr from Krane-managed deployment pods

CREATE TABLE IF NOT EXISTS default.runtime_logs_raw_v1
(
    `time` Int64 CODEC(Delta, LZ4),
    `severity` LowCardinality(String),
    `message` String CODEC(ZSTD(1)),
    `workspace_id` String CODEC(ZSTD(1)),
    `project_id` String CODEC(ZSTD(1)),
    `environment_id` String CODEC(ZSTD(1)),
    `deployment_id` String CODEC(ZSTD(1)),
    `k8s_pod_name` String CODEC(ZSTD(1)),
    `region` LowCardinality(String),
    `attributes` Map(String, String) CODEC(ZSTD(1)),
    INDEX idx_workspace_id workspace_id TYPE bloom_filter GRANULARITY 1,
    INDEX idx_deployment_id deployment_id TYPE bloom_filter GRANULARITY 1,
    INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(time))
ORDER BY (workspace_id, deployment_id, time)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;
