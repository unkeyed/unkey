-- MCP tool calls: one row for every tools/call that svc/mcp (mcp.unkey.com)
-- handles. Internal analytics only; no customer-facing query reads it.
-- Tool arguments and results are never stored because they can hold
-- plaintext keys and customer data.
--
-- workspace_id is set only for OAuth calls the API answered with 2xx. Root
-- key calls have none: join api_request_id to api_requests_raw_v2.request_id
-- for their workspace.

CREATE TABLE IF NOT EXISTS default.mcp_tool_calls_raw_v1
(
    `request_id`     String CODEC(ZSTD(1)),
    `time`           Int64 CODEC(Delta, ZSTD(1)),
    -- 'v2' | 'compute'.
    `toolset`        LowCardinality(String),
    -- The URL path the client connected to. Legacy paths (/mcp/v2,
    -- /mcp/compute) show when the Speakeasy-era aliases can be retired.
    `mount`          LowCardinality(String),
    `tool`           LowCardinality(String),
    -- 'ok' | 'api_error' | 'transport_error' | 'rejected'.
    `outcome`        LowCardinality(String),
    -- API response status. Zero when no response was received.
    `http_status`    Int32,
    -- The API error's type URI. Empty on success.
    `error_type`     LowCardinality(String),
    `latency_ms`     Int64,
    -- 'oauth' | 'root_key'.
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
