CREATE TABLE frontline_requests_anomaly_per_5m_v1 (
  time DateTime,
  workspace_id String,
  anomaly_shard UInt8 MATERIALIZED cityHash64(workspace_id) % 256,
  project_id String,
  app_id String,
  environment_id String,
  error_5xx SimpleAggregateFunction(sum, Int64),
  error_4xx SimpleAggregateFunction(sum, Int64),
  requests SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (anomaly_shard, workspace_id, project_id, app_id, environment_id, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW frontline_requests_anomaly_per_5m_mv_v1
TO frontline_requests_anomaly_per_5m_v1 AS
SELECT
  time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  sumIf(count, response_status >= 500 AND response_status < 600) AS error_5xx,
  sumIf(count, response_status >= 400 AND response_status < 500) AS error_4xx,
  sum(count) AS requests
FROM frontline_requests_per_5m_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id;

INSERT INTO frontline_requests_anomaly_per_5m_v1
SELECT
  time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  sumIf(count, response_status >= 500 AND response_status < 600) AS error_5xx,
  sumIf(count, response_status >= 400 AND response_status < 500) AS error_4xx,
  sum(count) AS requests
FROM frontline_requests_per_5m_v1
WHERE time >= now() - INTERVAL 7 DAY
  AND (workspace_id, project_id, app_id, environment_id, time) NOT IN (
    SELECT workspace_id, project_id, app_id, environment_id, time
    FROM frontline_requests_anomaly_per_5m_v1
    WHERE time >= now() - INTERVAL 7 DAY
  )
GROUP BY time, workspace_id, project_id, app_id, environment_id;
