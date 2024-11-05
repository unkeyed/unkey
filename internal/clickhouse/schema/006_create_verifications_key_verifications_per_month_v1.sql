-- +goose up
CREATE TABLE verifications.key_verifications_per_month_v1
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, key_space_id, time, identity_id, key_id)
;

-- +goose down
DROP TABLE verifications.key_verifications_per_month_v1;
