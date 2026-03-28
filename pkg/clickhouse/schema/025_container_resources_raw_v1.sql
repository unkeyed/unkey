CREATE TABLE container_resource_snapshots_v1 (
  time DateTime,
  workspace_id String,
  project_id String,
  app_id String,
  environment_id String,
  deployment_id String,
  instance_id String,
  region LowCardinality(String),
  platform LowCardinality(String),

  -- Actual usage (from Metrics Server)
  cpu_millicores Int32,
  memory_bytes Int64,

  -- Allocated resources (from pod spec via informer)
  cpu_request_millicores Int32,
  cpu_limit_millicores Int32,
  memory_request_bytes Int64,
  memory_limit_bytes Int64,

  -- Network egress (from Hubble)
  network_egress_bytes Int64,
  network_egress_public_bytes Int64
)
ENGINE = ReplacingMergeTree(time)
ORDER BY (workspace_id, deployment_id, instance_id, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 90 DAY DELETE;
