-- +goose up
CREATE TABLE ratelimits.ratelimits_per_day_v1
(
  time          DateTime,
  workspace_id  String,
  namespace_id  String,
  identifier    String,

  pass          Int8,
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier, pass)
;
