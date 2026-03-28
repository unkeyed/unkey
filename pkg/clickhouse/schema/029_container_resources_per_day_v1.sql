CREATE TABLE container_resources_per_day_v1 (
  time Date,
  workspace_id String,
  project_id String,
  app_id String,
  environment_id String,
  deployment_id String,
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
ORDER BY (workspace_id, deployment_id, instance_id, time)
TTL time + INTERVAL 365 DAY DELETE;

CREATE MATERIALIZED VIEW container_resources_per_day_mv_v1
TO container_resources_per_day_v1 AS
SELECT
  toStartOfDay(time) AS time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  deployment_id,
  instance_id,
  sum(cpu_millicores_sum) AS cpu_millicores_sum,
  max(memory_bytes_max) AS memory_bytes_max,
  sum(memory_bytes_sum) AS memory_bytes_sum,
  max(cpu_limit_millicores_max) AS cpu_limit_millicores_max,
  max(memory_limit_bytes_max) AS memory_limit_bytes_max,
  sum(network_egress_bytes_sum) AS network_egress_bytes_sum,
  sum(network_egress_public_bytes_sum) AS network_egress_public_bytes_sum,
  sum(sample_count) AS sample_count
FROM container_resources_per_hour_v1
GROUP BY
  time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  deployment_id,
  instance_id;
