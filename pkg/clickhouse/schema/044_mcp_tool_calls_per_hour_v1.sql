CREATE TABLE IF NOT EXISTS default.mcp_tool_calls_per_hour_v1
(
    `time`        DateTime,
    `toolset`     LowCardinality(String),
    `mount`       LowCardinality(String),
    `tool`        LowCardinality(String),
    `auth_kind`   LowCardinality(String),
    `outcome`     LowCardinality(String),
    `http_status` Int32,
    `client_name` LowCardinality(String),
    `count`       SimpleAggregateFunction(sum, Int64),
    `workspaces`  AggregateFunction(uniqIf, String, UInt8),
    `latency_p50` AggregateFunction(quantileTDigest(0.5), Float64),
    `latency_p95` AggregateFunction(quantileTDigest(0.95), Float64),
    `latency_p99` AggregateFunction(quantileTDigest(0.99), Float64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (toolset, tool, time, mount, auth_kind, outcome, http_status, client_name)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 400 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.mcp_tool_calls_per_hour_mv_v1
TO default.mcp_tool_calls_per_hour_v1 AS
SELECT
  toStartOfHour(fromUnixTimestamp64Milli(time)) AS time,
  toolset,
  mount,
  tool,
  auth_kind,
  outcome,
  http_status,
  client_name,
  toInt64(count()) AS count,
  uniqIfState(workspace_id, workspace_id != '') AS workspaces,
  quantileTDigestState(0.5)(CAST(latency_ms AS Float64)) AS latency_p50,
  quantileTDigestState(0.95)(CAST(latency_ms AS Float64)) AS latency_p95,
  quantileTDigestState(0.99)(CAST(latency_ms AS Float64)) AS latency_p99
FROM default.mcp_tool_calls_raw_v1
GROUP BY time, toolset, mount, tool, auth_kind, outcome, http_status, client_name;
