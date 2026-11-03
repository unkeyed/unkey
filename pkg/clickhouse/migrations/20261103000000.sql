-- MCP tool calls written by svc/mcp, with an hourly rollup. Internal
-- analytics only. See schema/043_mcp_tool_calls_raw_v1.sql and
-- schema/044_mcp_tool_calls_per_hour_v1.sql.

CREATE TABLE `default`.`mcp_tool_calls_raw_v1` (
  `request_id` String CODEC(ZSTD(1)),
  `time` Int64 CODEC(Delta(8), ZSTD(1)),
  `toolset` LowCardinality(String),
  `mount` LowCardinality(String),
  `tool` LowCardinality(String),
  `outcome` LowCardinality(String),
  `http_status` Int32,
  `error_type` LowCardinality(String),
  `latency_ms` Int64,
  `auth_kind` LowCardinality(String),
  `workspace_id` String CODEC(ZSTD(1)),
  `session_id` String CODEC(ZSTD(1)),
  `api_request_id` String CODEC(ZSTD(1)),
  `client_name` LowCardinality(String),
  `client_version` LowCardinality(String),
  INDEX `idx_workspace_id` ((`workspace_id`)) TYPE bloom_filter GRANULARITY 4,
  INDEX `idx_api_request_id` ((`api_request_id`)) TYPE bloom_filter GRANULARITY 4
) ENGINE = MergeTree
PARTITION BY (toYYYYMM(fromUnixTimestamp64Milli(`time`)))
PRIMARY KEY (`toolset`, `tool`, `time`) ORDER BY (`toolset`, `tool`, `time`)
TTL toDateTime(fromUnixTimestamp64Milli(`time`)) + toIntervalDay(90)
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000, ttl_only_drop_parts = 1;

CREATE TABLE `default`.`mcp_tool_calls_per_hour_v1` (
  `time` DateTime,
  `toolset` LowCardinality(String),
  `mount` LowCardinality(String),
  `tool` LowCardinality(String),
  `auth_kind` LowCardinality(String),
  `outcome` LowCardinality(String),
  `http_status` Int32,
  `client_name` LowCardinality(String),
  `count` SimpleAggregateFunction(sum, Int64),
  `workspaces` AggregateFunction(uniqIf, String, UInt8),
  `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
  `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
  `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
) ENGINE = AggregatingMergeTree
PARTITION BY (toYYYYMM(`time`))
PRIMARY KEY (`toolset`, `tool`, `time`, `mount`, `auth_kind`, `outcome`, `http_status`, `client_name`) ORDER BY (`toolset`, `tool`, `time`, `mount`, `auth_kind`, `outcome`, `http_status`, `client_name`)
TTL `time` + toIntervalDay(400)
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW `default`.`mcp_tool_calls_per_hour_mv_v1`
TO `default`.`mcp_tool_calls_per_hour_v1` AS
SELECT
  toStartOfHour(fromUnixTimestamp64Milli(`time`)) AS `time`,
  `toolset`,
  `mount`,
  `tool`,
  `auth_kind`,
  `outcome`,
  `http_status`,
  `client_name`,
  toInt64(count()) AS `count`,
  uniqIfState(`workspace_id`, `workspace_id` != '') AS `workspaces`,
  quantileTDigestState(0.5)(CAST(`latency_ms` AS Float64)) AS `latency_p50`,
  quantileTDigestState(0.95)(CAST(`latency_ms` AS Float64)) AS `latency_p95`,
  quantileTDigestState(0.99)(CAST(`latency_ms` AS Float64)) AS `latency_p99`
FROM `default`.`mcp_tool_calls_raw_v1`
GROUP BY
  `time`,
  `toolset`,
  `mount`,
  `tool`,
  `auth_kind`,
  `outcome`,
  `http_status`,
  `client_name`;
