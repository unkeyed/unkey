CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_last_used_v1
(
  time          Int64,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;
