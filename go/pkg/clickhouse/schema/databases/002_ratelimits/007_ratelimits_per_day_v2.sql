CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_per_day_v2
(
  time          DateTime,
  workspace_id  String,
  namespace_id  String,
  identifier    String,
  passed        Int64,
  total         Int64,
  latency_avg   Float64,
  latency_p75   Float64,
  latency_p99   Float64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, time, namespace_id, identifier)
;