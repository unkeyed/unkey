-- +goose up
CREATE TABLE metrics.api_requests_per_minute_v1
(
  workspace_id  String,
  identifier    String,
  time          SimpleAggregateFunction(max, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, namespace_id, identifier)
;

-- +goose down
DROP TABLE ratelimits.ratelimits_last_used_v1;
