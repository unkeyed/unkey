CREATE TABLE IF NOT EXISTS ratelimits.ratelimits_last_used_v2
(
  time          Int64,
  workspace_id  String,
  namespace_id  String,
  identifier    String

)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, time, namespace_id, identifier)
;
