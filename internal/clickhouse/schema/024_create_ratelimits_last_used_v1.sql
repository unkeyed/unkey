-- +goose up
CREATE TABLE ratelimits.ratelimits_last_used_v1
(
  workspace_id  String,
  namespace_id  String,
  identifier    String,
  time          SimpleAggregateFunction(max, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, namespace_id, identifier)
;

