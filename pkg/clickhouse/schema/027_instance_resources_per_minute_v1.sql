CREATE TABLE instance_resources_per_minute_v1 (
  time DateTime,
  workspace_id String,
  project_id String,
  app_id String,
  environment_id String,
  resource_type LowCardinality(String),
  resource_id String,
  instance_id String,
  cpu_millicores_sum SimpleAggregateFunction(sum, Int64),
  memory_bytes_max SimpleAggregateFunction(max, Int64),
  memory_bytes_sum SimpleAggregateFunction(sum, Int64),
  cpu_limit_millicores_max SimpleAggregateFunction(max, Int32),
  memory_limit_bytes_max SimpleAggregateFunction(max, Int64),
  network_egress_bytes_sum SimpleAggregateFunction(sum, Int64),
  network_egress_public_bytes_sum SimpleAggregateFunction(sum, Int64),
  sample_count SimpleAggregateFunction(sum, Int64)
) ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, instance_id, time)
PARTITION BY toStartOfDay(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW instance_resources_per_minute_mv_v1
TO instance_resources_per_minute_v1 AS
SELECT
  toStartOfMinute(time) AS time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  resource_type,
  resource_id,
  instance_id,
  sum(toInt64(cpu_millicores)) AS cpu_millicores_sum,
  max(memory_bytes) AS memory_bytes_max,
  sum(memory_bytes) AS memory_bytes_sum,
  max(cpu_limit_millicores) AS cpu_limit_millicores_max,
  max(memory_limit_bytes) AS memory_limit_bytes_max,
  sum(network_egress_bytes) AS network_egress_bytes_sum,
  sum(network_egress_public_bytes) AS network_egress_public_bytes_sum,
  count() AS sample_count
FROM instance_resource_snapshots_v1
GROUP BY
  time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  resource_type,
  resource_id,
  instance_id;
