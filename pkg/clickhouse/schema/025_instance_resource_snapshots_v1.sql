CREATE TABLE instance_resource_snapshots_v1 (
  time DateTime,
  workspace_id String,
  project_id String,
  app_id String,
  environment_id String,
  resource_type LowCardinality(String),
  resource_id String,
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

  -- Network egress (from conntrack)
  network_egress_bytes Int64,
  network_egress_public_bytes Int64,

  -- Pod real start time from K8s (unix milliseconds)
  started_at Int64
)
ENGINE = ReplacingMergeTree(time)
ORDER BY (workspace_id, resource_id, instance_id, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 90 DAY DELETE;
