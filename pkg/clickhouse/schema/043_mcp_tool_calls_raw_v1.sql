CREATE TABLE IF NOT EXISTS default.mcp_tool_calls_raw_v1
(
    `request_id`     String CODEC(ZSTD(1)),
    `time`           Int64 CODEC(Delta, ZSTD(1)),
    `toolset`        LowCardinality(String),
    `mount`          LowCardinality(String),
    `tool`           LowCardinality(String),
    `outcome`        LowCardinality(String),
    `http_status`    Int32,
    `error_type`     LowCardinality(String),
    `latency_ms`     Int64,
    `auth_kind`      LowCardinality(String),
    `workspace_id`   String CODEC(ZSTD(1)),
    `session_id`     String CODEC(ZSTD(1)),
    `api_request_id` String CODEC(ZSTD(1)),
    `client_name`    LowCardinality(String),
    `client_version` LowCardinality(String),

    INDEX idx_workspace_id workspace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_api_request_id api_request_id TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(time))
ORDER BY (toolset, tool, time)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 10000,
         ttl_only_drop_parts = 1;
