CREATE MATERIALIZED VIEW IF NOT EXISTS verifications.key_verifications_per_minute_mv_v2
TO verifications.key_verifications_per_minute_v2
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  sum(spent_credits) as spent_credits,
  avg(latency) as latency_avg,
  quantileTDigest(0.75)(latency) as latency_p75,
  quantileTDigest(0.99)(latency) as latency_p99,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
  tags
FROM verifications.raw_key_verifications_v2
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time,
  tags
;