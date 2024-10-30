-- +goose up
CREATE TABLE default.key_verifications_per_day_v1
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  count         AggregateFunction(count, UInt64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, key_space_id, time, identity_id, key_id)
;
