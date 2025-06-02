-- +goose up
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_minute_mv_v1
TO verifications.key_verifications_per_minute_v1
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v1
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;

-- +goose down
DROP VIEW verifications.key_verifications_per_minute_mv_v1;
