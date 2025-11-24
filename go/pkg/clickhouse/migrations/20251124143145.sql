ALTER TABLE `default`.`key_verifications_raw_v2` MODIFY TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(90);
-- Create "key_verifications_per_day_v3" table
CREATE TABLE `default`.`key_verifications_per_day_v3` (
  `time` Date,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `external_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` SimpleAggregateFunction(sum, Int64),
  `spent_credits` SimpleAggregateFunction(sum, Int64),
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) PARTITION BY (toStartOfMonth(time)) TTL time + toIntervalDay(356) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_hour_v3" table
CREATE TABLE `default`.`key_verifications_per_hour_v3` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `external_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` SimpleAggregateFunction(sum, Int64),
  `spent_credits` SimpleAggregateFunction(sum, Int64),
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) PARTITION BY (toStartOfDay(time)) TTL time + toIntervalDay(90) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_minute_v3" table
CREATE TABLE `default`.`key_verifications_per_minute_v3` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `external_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` SimpleAggregateFunction(sum, Int64),
  `spent_credits` SimpleAggregateFunction(sum, Int64),
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) PARTITION BY (toStartOfDay(time)) TTL time + toIntervalDay(90) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_month_v3" table
CREATE TABLE `default`.`key_verifications_per_month_v3` (
  `time` Date,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `external_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` SimpleAggregateFunction(sum, Int64),
  `spent_credits` SimpleAggregateFunction(sum, Int64),
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `external_id`, `key_id`, `outcome`, `tags`) PARTITION BY (toStartOfYear(time)) TTL time + toIntervalYear(3) SETTINGS index_granularity = 8192;
-- Drop "active_workspaces_keys_per_month_mv_v2" view
DROP VIEW `default`.`active_workspaces_keys_per_month_mv_v2`;
-- Create "active_workspaces_keys_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`active_workspaces_keys_per_month_mv_v2` TO `default`.`active_workspaces_per_month_v2` AS SELECT workspace_id, toDate(time) AS time FROM default.key_verifications_per_month_v3;
-- Drop "billable_verifications_per_month_mv_v2" view
DROP VIEW `default`.`billable_verifications_per_month_mv_v2`;
-- Create "billable_verifications_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`billable_verifications_per_month_mv_v2` TO `default`.`billable_verifications_per_month_v2` AS SELECT workspace_id, sum(count) AS count, toYear(time) AS year, toMonth(time) AS month FROM default.key_verifications_per_month_v3 WHERE outcome = 'VALID' GROUP BY workspace_id, year, month;
-- Create "key_verifications_per_day_mv_v3" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_day_mv_v3` TO `default`.`key_verifications_per_day_v3` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toDate(toStartOfDay(time)) AS time FROM default.key_verifications_per_hour_v3 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
-- Create "key_verifications_per_hour_mv_v3" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_hour_mv_v3` TO `default`.`key_verifications_per_hour_v3` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfHour(time) AS time FROM default.key_verifications_per_minute_v3 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
-- Create "key_verifications_per_minute_mv_v3" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_minute_mv_v3` TO `default`.`key_verifications_per_minute_v3` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, count(*) AS count, sum(spent_credits) AS spent_credits, avgState(latency) AS latency_avg, quantilesTDigestState(0.75)(latency) AS latency_p75, quantilesTDigestState(0.99)(latency) AS latency_p99, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM default.key_verifications_raw_v2 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
-- Create "key_verifications_per_month_mv_v3" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_month_mv_v3` TO `default`.`key_verifications_per_month_v3` AS SELECT workspace_id, key_space_id, identity_id, external_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toDate(toStartOfMonth(time)) AS time FROM default.key_verifications_per_day_v3 GROUP BY workspace_id, time, key_space_id, identity_id, external_id, key_id, outcome, tags;
