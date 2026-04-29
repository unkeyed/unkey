-- Track cost per ratelimit decision so we can chart spend for both
-- accepted and rejected standalone ratelimits (ENG-2767).
--
-- The raw table gets a `cost` column. Aggregated views get both
-- `total_cost` (sum of cost across all decisions) and `passed_cost`
-- (sum of cost for passed decisions only). Existing rows default to 0,
-- which is correct: pre-rollout we have no cost data to attribute.

ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `cost` UInt64 AFTER `reset_at`;

ALTER TABLE `default`.`ratelimits_per_minute_v2`
  ADD COLUMN `total_cost` SimpleAggregateFunction(sum, Int64) AFTER `total`,
  ADD COLUMN `passed_cost` SimpleAggregateFunction(sum, Int64) AFTER `total_cost`;

ALTER TABLE `default`.`ratelimits_per_hour_v2`
  ADD COLUMN `total_cost` SimpleAggregateFunction(sum, Int64) AFTER `total`,
  ADD COLUMN `passed_cost` SimpleAggregateFunction(sum, Int64) AFTER `total_cost`;

ALTER TABLE `default`.`ratelimits_per_day_v2`
  ADD COLUMN `total_cost` SimpleAggregateFunction(sum, Int64) AFTER `total`,
  ADD COLUMN `passed_cost` SimpleAggregateFunction(sum, Int64) AFTER `total_cost`;

ALTER TABLE `default`.`ratelimits_per_month_v2`
  ADD COLUMN `total_cost` SimpleAggregateFunction(sum, Int64) AFTER `total`,
  ADD COLUMN `passed_cost` SimpleAggregateFunction(sum, Int64) AFTER `total_cost`;

-- Replace the materialized views so they start populating cost columns.
-- DROP+CREATE is required: ALTER MATERIALIZED VIEW MODIFY QUERY would
-- preserve the schema mismatch with the new target columns.
--
-- The brief window between DROP and CREATE is a real but tiny aggregation
-- gap (sub-second in practice — atlas runs the statements back-to-back).
-- Raw events keep landing in `ratelimits_raw_v2` with the new `cost`
-- column, so if a backfill is ever needed run, e.g.:
--
--   INSERT INTO `default`.`ratelimits_per_minute_v2` SELECT ...
--   FROM `default`.`ratelimits_raw_v2`
--   WHERE time >= toUnixTimestamp64Milli('<migration_start>')
--     AND time <  toUnixTimestamp64Milli('<migration_end>')
--   GROUP BY workspace_id, namespace_id, identifier, time;

DROP VIEW IF EXISTS `default`.`ratelimits_per_minute_mv_v2`;
-- The `passed` alias shadows the column name, which makes ClickHouse
-- read the second arg of sumIf/countIf as another aggregate. Aliasing
-- the source table (`r`) lets us reference the underlying column
-- unambiguously.
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_minute_mv_v2` TO `default`.`ratelimits_per_minute_v2` AS
SELECT
  r.workspace_id AS workspace_id,
  r.namespace_id AS namespace_id,
  r.identifier AS identifier,
  count(*) AS total,
  countIf(r.passed > 0) AS passed,
  sum(r.cost) AS total_cost,
  sumIf(r.cost, r.passed > 0) AS passed_cost,
  avgState(r.latency) AS latency_avg,
  quantilesTDigestState(0.75)(r.latency) AS latency_p75,
  quantilesTDigestState(0.99)(r.latency) AS latency_p99,
  toStartOfMinute(fromUnixTimestamp64Milli(r.time)) AS time
FROM default.ratelimits_raw_v2 AS r
GROUP BY r.workspace_id, r.namespace_id, time, r.identifier;

DROP VIEW IF EXISTS `default`.`ratelimits_per_hour_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_hour_mv_v2` TO `default`.`ratelimits_per_hour_v2` AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  sum(total) AS total,
  sum(passed) AS passed,
  sum(total_cost) AS total_cost,
  sum(passed_cost) AS passed_cost,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toStartOfHour(time) AS time
FROM default.ratelimits_per_minute_v2
GROUP BY workspace_id, namespace_id, time, identifier;

DROP VIEW IF EXISTS `default`.`ratelimits_per_day_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_day_mv_v2` TO `default`.`ratelimits_per_day_v2` AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  sum(total) AS total,
  sum(passed) AS passed,
  sum(total_cost) AS total_cost,
  sum(passed_cost) AS passed_cost,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfDay(time)) AS time
FROM default.ratelimits_per_hour_v2
GROUP BY workspace_id, namespace_id, time, identifier;

DROP VIEW IF EXISTS `default`.`ratelimits_per_month_mv_v2`;
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_month_mv_v2` TO `default`.`ratelimits_per_month_v2` AS
SELECT
  workspace_id,
  namespace_id,
  identifier,
  sum(total) AS total,
  sum(passed) AS passed,
  sum(total_cost) AS total_cost,
  sum(passed_cost) AS passed_cost,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfMonth(time)) AS time
FROM default.ratelimits_per_day_v2
GROUP BY workspace_id, namespace_id, time, identifier;
