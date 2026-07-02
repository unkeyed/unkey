-- Add a billing source to key verifications (ENG-2892).
--
-- Unkey Deploy's gateway (frontline) verifies keys on behalf of deployed apps
-- through the same pipeline as the public API, so its usage was
-- indistinguishable from billable API usage. `source` tags every row with
-- where it originated: 'api' (the public API) or 'gateway' (the Deploy
-- key-auth policy). Billing rollups exclude 'gateway'; analytics rollups keep
-- both, since gateway traffic is still the customer's traffic.
--
-- Existing raw rows default to 'api', which is correct: everything written
-- before frontline tagged its rows came from the API. The default also makes
-- billable the safe direction; a writer must explicitly opt out of billing.
--
-- `source` is a dimension, not an aggregate, so unlike the tokens migration
-- (20260429000000) the rollup tables also append it to ORDER BY: rows with
-- different sources must not merge into one another in AggregatingMergeTree.
-- Appending a newly added column to the end of ORDER BY is a metadata-only
-- ALTER; the PRIMARY KEY stays a prefix of the new sorting key. MODIFY
-- ORDER BY only accepts columns added in the same ALTER *without* a DEFAULT
-- (verified against ClickHouse Cloud), so the rollup columns have none:
-- pre-migration rollup rows read source = '', which the billable filters
-- (source != 'gateway') still count as billable, same as 'api'.
--
-- The views are swapped with ALTER TABLE ... MODIFY QUERY, which is atomic:
-- no ingestion gap, unlike the DROP+CREATE the tokens migration needed (its
-- target tables gained aggregate columns; ours only gain a plain dimension,
-- which MODIFY QUERY handles).
--
-- ROLLOUT ORDER: every writer binary (api, frontline) must be FULLY rolled
-- out before this migration runs. Writers insert over the native protocol
-- with the server's full column list; a binary whose row struct lacks
-- `source` fails every insert once the column exists ("missing destination
-- name"), dropping verification events for the rest of the rolling window.
-- The reverse order is safe: a new binary inserting into the old schema
-- simply has its source field ignored until the column appears. Rows written
-- in that window read source = 'api', which is the pre-migration status quo.

-- ----------------------------------------------------------------------
-- Key verifications
-- ----------------------------------------------------------------------

ALTER TABLE `default`.`key_verifications_raw_v2`
  ADD COLUMN `source` LowCardinality(String) DEFAULT 'api' AFTER `region`;

ALTER TABLE `default`.`key_verifications_per_minute_v3`
  ADD COLUMN `source` LowCardinality(String) AFTER `outcome`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`);

ALTER TABLE `default`.`key_verifications_per_hour_v3`
  ADD COLUMN `source` LowCardinality(String) AFTER `outcome`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`);

ALTER TABLE `default`.`key_verifications_per_day_v3`
  ADD COLUMN `source` LowCardinality(String) AFTER `outcome`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`);

ALTER TABLE `default`.`key_verifications_per_month_v3`
  ADD COLUMN `source` LowCardinality(String) AFTER `outcome`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`);

ALTER TABLE `default`.`key_verifications_per_minute_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  tags,
  count(*) AS count,
  sum(spent_credits) AS spent_credits,
  avgState(latency) AS latency_avg,
  quantilesTDigestState(0.75)(latency) AS latency_p75,
  quantilesTDigestState(0.99)(latency) AS latency_p99,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
FROM default.key_verifications_raw_v2
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, tags;

ALTER TABLE `default`.`key_verifications_per_hour_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  tags,
  sum(count) AS count,
  sum(spent_credits) AS spent_credits,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toStartOfHour(time) AS time
FROM default.key_verifications_per_minute_v3
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, tags;

ALTER TABLE `default`.`key_verifications_per_day_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  tags,
  sum(count) AS count,
  sum(spent_credits) AS spent_credits,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfDay(time)) AS time
FROM default.key_verifications_per_hour_v3
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, tags;

ALTER TABLE `default`.`key_verifications_per_month_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  tags,
  sum(count) AS count,
  sum(spent_credits) AS spent_credits,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfMonth(time)) AS time
FROM default.key_verifications_per_day_v3
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, tags;

-- Billing: same target table, the view adds the source filter. Gateway
-- verifications are not billed as API usage (Deploy is metered separately).
ALTER TABLE `default`.`billable_verifications_per_month_mv_v2` MODIFY QUERY
SELECT
  workspace_id,
  sum(count) AS count,
  toYear(time) AS year,
  toMonth(time) AS month
FROM default.key_verifications_per_month_v3
WHERE outcome = 'VALID' AND source != 'gateway'
GROUP BY workspace_id, year, month;
