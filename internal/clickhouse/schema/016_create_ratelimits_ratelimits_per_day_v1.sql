-- +goose up
CREATE TABLE ratelimits.ratelimits_per_day_v1
(
  time          DateTime,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

  passed        Int64,
  total         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier)
;



-- +goose down
DROP TABLE ratelimits.ratelimits_per_day_v1;
