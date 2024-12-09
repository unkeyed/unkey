-- +goose up
CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_month_mv_v3
TO verifications.key_verifications_per_month_v3
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
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

-- populate from existing data
-- INSERT INTO verifications.key_verifications_per_month_v3
-- SELECT
--   toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time,
--   workspace_id,
--   key_space_id,
--   identity_id,
--   key_id,
--   outcome,
--   tags,
--   count(*) as counts
-- FROM verifications.raw_key_verifications_v1
-- GROUP BY
--   workspace_id,
--   key_space_id,
--   identity_id,
--   key_id,
--   outcome,
--   time,
--   tags
-- ;


-- +goose down
DROP VIEW verifications.key_verifications_per_month_mv_v3;
