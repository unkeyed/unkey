-- Attribute Gateway key verifications to the app that ran KeyAuth.
--
-- `source = 'gateway'` tells us that Gateway performed a verification, but a
-- key space can be used by more than one app. `app_id` adds that missing
-- dimension. API and historical verifications use the empty string.
--
-- `app_id` is appended to each sorting key instead of preceding `time` so
-- common time-bounded queries keep primary-key pruning. Appending the newly
-- added dimension also keeps these ALTERs metadata-only.
--
-- DEPLOYMENT ORDER: apply this migration before deploying writers that include
-- `app_id` in their explicit insert column list. Old writers remain compatible
-- because the raw column has a server-side default.

ALTER TABLE `default`.`key_verifications_raw_v2`
  ADD COLUMN `app_id` LowCardinality(String) DEFAULT '' AFTER `source`;

ALTER TABLE `default`.`key_verifications_per_minute_v3`
  ADD COLUMN `app_id` LowCardinality(String) AFTER `source`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`, `app_id`);

ALTER TABLE `default`.`key_verifications_per_hour_v3`
  ADD COLUMN `app_id` LowCardinality(String) AFTER `source`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`, `app_id`);

ALTER TABLE `default`.`key_verifications_per_day_v3`
  ADD COLUMN `app_id` LowCardinality(String) AFTER `source`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`, `app_id`);

ALTER TABLE `default`.`key_verifications_per_month_v3`
  ADD COLUMN `app_id` LowCardinality(String) AFTER `source`,
  MODIFY ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`, `source`, `app_id`);

ALTER TABLE `default`.`key_verifications_per_minute_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  app_id,
  tags,
  count(*) AS count,
  sum(spent_credits) AS spent_credits,
  avgState(latency) AS latency_avg,
  quantilesTDigestState(0.75)(latency) AS latency_p75,
  quantilesTDigestState(0.99)(latency) AS latency_p99,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
FROM default.key_verifications_raw_v2
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, app_id, tags;

ALTER TABLE `default`.`key_verifications_per_hour_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  app_id,
  tags,
  sum(count) AS count,
  sum(spent_credits) AS spent_credits,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toStartOfHour(time) AS time
FROM default.key_verifications_per_minute_v3
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, app_id, tags;

ALTER TABLE `default`.`key_verifications_per_day_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  app_id,
  tags,
  sum(count) AS count,
  sum(spent_credits) AS spent_credits,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfDay(time)) AS time
FROM default.key_verifications_per_hour_v3
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, app_id, tags;

ALTER TABLE `default`.`key_verifications_per_month_mv_v3` MODIFY QUERY
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  source,
  app_id,
  tags,
  sum(count) AS count,
  sum(spent_credits) AS spent_credits,
  avgMergeState(latency_avg) AS latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99,
  toDate(toStartOfMonth(time)) AS time
FROM default.key_verifications_per_day_v3
GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, source, app_id, tags;
