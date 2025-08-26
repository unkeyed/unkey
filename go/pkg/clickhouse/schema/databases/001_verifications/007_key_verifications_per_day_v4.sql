CREATE TABLE IF NOT EXISTS verifications.key_verifications_per_day_v4
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64,
  spent_credits Int64,
  latency_avg   Float64,
  latency_p75   Float64,
  latency_p99   Float64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, time, key_space_id, identity_id, key_id, tags, outcome)
;