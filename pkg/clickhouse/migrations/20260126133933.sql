-- Runtime logs table for customer deployment logs
-- Stores stdout/stderr from Krane-managed deployment pods

CREATE TABLE IF NOT EXISTS default.runtime_logs_raw_v1
(
    `time` Int64 CODEC(Delta, LZ4),
    `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)),
    `severity` LowCardinality(String),
    `message` String CODEC(ZSTD(1)),
    `workspace_id` String CODEC(ZSTD(1)),
    `project_id` String CODEC(ZSTD(1)),
    `environment_id` String CODEC(ZSTD(1)),
    `deployment_id` String CODEC(ZSTD(1)),
    `k8s_pod_name` String CODEC(ZSTD(1)),
    `region` LowCardinality(String),
    `attributes` JSON CODEC(ZSTD(1)),
    `attributes_text` String MATERIALIZED toJSONString(attributes) CODEC(ZSTD(1)),
    `expires_at` DateTime64(3) DEFAULT now64(3) + INTERVAL 90 DAY,
    INDEX idx_workspace_id workspace_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_deployment_id deployment_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1,
    INDEX idx_attributes_text attributes_text TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toDate(fromUnixTimestamp64Milli(inserted_at))
ORDER BY (workspace_id, project_id, environment_id, time, deployment_id)
TTL expires_at + INTERVAL 7 DAY
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;
