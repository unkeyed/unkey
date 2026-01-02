CREATE TABLE build_step_logs_v1
(
  -- unix milli
  time Int64 CODEC(Delta, LZ4),

  workspace_id String,
  project_id String,
  deployment_id String,

  message String,
  step_id String,
)
ENGINE = MergeTree()
ORDER BY (workspace_id, project_id, deployment_id, step_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 3 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000
;



