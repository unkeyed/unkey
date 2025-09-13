-- +goose up
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_month_mv_v4
TO verifications.key_verifications_per_month_v4
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  sum(spent_credits) as spent_credits,
  toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time,
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
DROP VIEW IF EXISTS verifications.key_verifications_per_month_mv_v4;
