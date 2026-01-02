-- Drop "key_verifications_per_month_mv_v2" view
DROP VIEW `default`.`key_verifications_per_month_mv_v2`;
ALTER TABLE `default`.`key_verifications_per_day_v2` ADD COLUMN `external_id` String;
-- Drop "key_verifications_per_day_mv_v2" view
DROP VIEW `default`.`key_verifications_per_day_mv_v2`;
ALTER TABLE `default`.`key_verifications_per_hour_v2` ADD COLUMN `external_id` String;
-- Drop "key_verifications_per_hour_mv_v2" view
DROP VIEW `default`.`key_verifications_per_hour_mv_v2`;
ALTER TABLE `default`.`key_verifications_per_minute_v2` ADD COLUMN `external_id` String;
ALTER TABLE `default`.`key_verifications_per_month_v2` ADD COLUMN `external_id` String;
-- Drop "key_verifications_per_minute_mv_v2" view
DROP VIEW `default`.`key_verifications_per_minute_mv_v2`;
ALTER TABLE `default`.`key_verifications_raw_v2` ADD COLUMN `external_id` String;
ALTER TABLE `default`.`key_verifications_raw_v2` ADD INDEX `idx_external_id` ((external_id)) TYPE bloom_filter GRANULARITY 1;
-- Create "key_verifications_per_day_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_day_mv_v2` TO `default`.`key_verifications_per_day_v2` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toDate(toStartOfDay(time)) AS time FROM default.key_verifications_per_hour_v2 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
-- Create "key_verifications_per_hour_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_hour_mv_v2` TO `default`.`key_verifications_per_hour_v2` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfHour(time) AS time FROM default.key_verifications_per_minute_v2 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
-- Create "key_verifications_per_minute_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_minute_mv_v2` TO `default`.`key_verifications_per_minute_v2` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, count(*) AS count, sum(spent_credits) AS spent_credits, avgState(latency) AS latency_avg, quantilesTDigestState(0.75)(latency) AS latency_p75, quantilesTDigestState(0.99)(latency) AS latency_p99, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM default.key_verifications_raw_v2 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
-- Create "key_verifications_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_month_mv_v2` TO `default`.`key_verifications_per_month_v2` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toDate(toStartOfMonth(time)) AS time FROM default.key_verifications_per_day_v2 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
-- Drop "temp_sync_key_verifications_v1_to_v2" view
DROP VIEW `default`.`temp_sync_key_verifications_v1_to_v2`;
-- Create "temp_sync_key_verifications_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_key_verifications_v1_to_v2` TO `default`.`key_verifications_raw_v2` AS SELECT request_id, time, workspace_id, key_space_id, identity_id, '' AS external_id, key_id, region, outcome, tags, 0 AS spent_credits, 0. AS latency FROM verifications.raw_key_verifications_v1;
