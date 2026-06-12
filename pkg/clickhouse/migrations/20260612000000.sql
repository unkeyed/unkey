-- Add a billing source to key verifications and ratelimits (ENG-2892).
--
-- Unkey Deploy's gateway (frontline) verifies keys and applies ratelimits on
-- behalf of deployed apps through the same pipeline as the public API, so its
-- usage was indistinguishable from billable API usage. `source` tags every
-- row with where it originated: 'api' (the public API) or 'gateway' (the
-- Deploy key-auth/ratelimit policies). Billing rollups exclude 'gateway';
-- analytics rollups keep both, since gateway traffic is still the customer's
-- traffic.
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
-- The brief window between each DROP VIEW and CREATE VIEW is a real but tiny
-- aggregation gap (sub-second; atlas runs the statements back-to-back). Raw
-- events keep landing in the raw tables throughout.

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

DROP VIEW IF EXISTS `default`.`key_verifications_per_minute_mv_v3`;
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_minute_mv_v3` TO `default`.`key_verifications_per_minute_v3` AS
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

DROP VIEW IF EXISTS `default`.`key_verifications_per_hour_mv_v3`;
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_hour_mv_v3` TO `default`.`key_verifications_per_hour_v3` AS
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

DROP VIEW IF EXISTS `default`.`key_verifications_per_day_mv_v3`;
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_day_mv_v3` TO `default`.`key_verifications_per_day_v3` AS
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

DROP VIEW IF EXISTS `default`.`key_verifications_per_month_mv_v3`;
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_month_mv_v3` TO `default`.`key_verifications_per_month_v3` AS
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
DROP VIEW IF EXISTS `default`.`billable_verifications_per_month_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`billable_verifications_per_month_mv_v2` TO `default`.`billable_verifications_per_month_v2` AS
SELECT
  workspace_id,
  sum(count) AS count,
  toYear(time) AS year,
  toMonth(time) AS month
FROM default.key_verifications_per_month_v3
WHERE outcome = 'VALID' AND source != 'gateway'
GROUP BY workspace_id, year, month;

-- ----------------------------------------------------------------------
-- Ratelimits
-- ----------------------------------------------------------------------

ALTER TABLE `default`.`ratelimits_raw_v2`
  ADD COLUMN `source` LowCardinality(String) DEFAULT 'api' AFTER `tokens`;

ALTER TABLE `default`.`ratelimits_per_minute_v2`
  ADD COLUMN `source` LowCardinality(String) AFTER `identifier`,
  MODIFY ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`, `source`);

ALTER TABLE `default`.`ratelimits_per_hour_v2`
  ADD COLUMN `source` LowCardinality(String) AFTER `identifier`,
  MODIFY ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`, `source`);

ALTER TABLE `default`.`ratelimits_per_day_v2`
  ADD COLUMN `source` LowCardinality(String) AFTER `identifier`,
  MODIFY ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`, `source`);

ALTER TABLE `default`.`ratelimits_per_month_v2`
  ADD COLUMN `source` LowCardinality(String) AFTER `identifier`,
  MODIFY ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`, `source`);

DROP VIEW IF EXISTS `default`.`ratelimits_per_minute_mv_v2`;
-- The `r` alias mirrors 20260429000000: the `passed` SELECT alias shadows
-- the column name otherwise.
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_minute_mv_v2` TO `default`.`ratelimits_per_minute_v2` AS
SELECT
  r.workspace_id AS workspace_id,
  r.namespace_id AS namespace_id,
  r.identifier AS identifier,
  r.source AS source,
  count(*) AS total,
  countIf(r.passed > 0) AS passed,
  sum(r.tokens) AS total_tokens,
  sumIf(r.tokens, r.passed > 0) AS passed_tokens,
  avgState(r.latency) AS latency_avg,
  quantilesTDigestState(0.75)(r.latency) AS latency_p75,
  quantilesTDigestState(0.99)(r.latency) AS latency_p99,
  toStartOfMinute(fromUnixTimestamp64Milli(r.time)) AS time
FROM default.ratelimits_raw_v2 AS r
GROUP BY r.workspace_id, r.namespace_id, time, r.identifier, r.source;

DROP VIEW IF EXISTS `default`.`ratelimits_per_hour_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_hour_mv_v2` TO `default`.`ratelimits_per_hour_v2` AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  source,
  sum(total) AS total,
  sum(passed) AS passed,
  sum(total_tokens) AS total_tokens,
  sum(passed_tokens) AS passed_tokens,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toStartOfHour(time) AS time
FROM default.ratelimits_per_minute_v2
GROUP BY workspace_id, namespace_id, time, identifier, source;

DROP VIEW IF EXISTS `default`.`ratelimits_per_day_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_day_mv_v2` TO `default`.`ratelimits_per_day_v2` AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  source,
  sum(total) AS total,
  sum(passed) AS passed,
  sum(total_tokens) AS total_tokens,
  sum(passed_tokens) AS passed_tokens,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfDay(time)) AS time
FROM default.ratelimits_per_hour_v2
GROUP BY workspace_id, namespace_id, time, identifier, source;

DROP VIEW IF EXISTS `default`.`ratelimits_per_month_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_month_mv_v2` TO `default`.`ratelimits_per_month_v2` AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  source,
  sum(total) AS total,
  sum(passed) AS passed,
  sum(total_tokens) AS total_tokens,
  sum(passed_tokens) AS passed_tokens,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfMonth(time)) AS time
FROM default.ratelimits_per_day_v2
GROUP BY workspace_id, namespace_id, time, identifier, source;

-- Billing: same target table, the view adds the source filter.
DROP VIEW IF EXISTS `default`.`billable_ratelimits_per_month_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`billable_ratelimits_per_month_mv_v2` TO `default`.`billable_ratelimits_per_month_v2` AS
SELECT
  workspace_id,
  sum(passed) AS count,
  toYear(time) AS year,
  toMonth(time) AS month
FROM default.ratelimits_per_month_v2
WHERE source != 'gateway'
GROUP BY workspace_id, year, month;
