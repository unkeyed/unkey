-- Add new schema named "billing"
CREATE DATABASE IF NOT EXISTS `billing` ENGINE Atomic;
-- Add new schema named "business"
CREATE DATABASE IF NOT EXISTS `business` ENGINE Atomic;
-- Add new schema named "metrics"
CREATE DATABASE IF NOT EXISTS `metrics` ENGINE Atomic;
-- Add new schema named "ratelimits"
CREATE DATABASE IF NOT EXISTS `ratelimits` ENGINE Atomic;
-- Add new schema named "verifications"
CREATE DATABASE IF NOT EXISTS `verifications` ENGINE Atomic;
-- Create "billable_ratelimits_per_month_v1" table
CREATE TABLE `billing`.`billable_ratelimits_per_month_v1` (
  `year` Int32,
  `month` Int32,
  `workspace_id` String,
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `year`, `month`) ORDER BY (`workspace_id`, `year`, `month`) SETTINGS index_granularity = 8192;
-- Create "billable_verifications_per_month_v1" table
CREATE TABLE `billing`.`billable_verifications_per_month_v1` (
  `year` Int32,
  `month` Int32,
  `workspace_id` String,
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `year`, `month`) ORDER BY (`workspace_id`, `year`, `month`) SETTINGS index_granularity = 8192;
-- Create "billable_verifications_per_month_v2" table
CREATE TABLE `billing`.`billable_verifications_per_month_v2` (
  `year` Int32,
  `month` Int32,
  `workspace_id` String,
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `year`, `month`) ORDER BY (`workspace_id`, `year`, `month`) SETTINGS index_granularity = 8192;
-- Create "active_workspaces_per_month_v1" table
CREATE TABLE `business`.`active_workspaces_per_month_v1` (
  `time` Date,
  `workspace_id` String
) ENGINE = MergeTree
PRIMARY KEY (`time`) ORDER BY (`time`) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_day_v1" table
CREATE TABLE `metrics`.`api_requests_per_day_v1` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) ORDER BY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_hour_v1" table
CREATE TABLE `metrics`.`api_requests_per_hour_v1` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) ORDER BY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_minute_v1" table
CREATE TABLE `metrics`.`api_requests_per_minute_v1` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) ORDER BY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) SETTINGS index_granularity = 8192;
-- Create "raw_api_requests_v1" table
CREATE TABLE `metrics`.`raw_api_requests_v1` (
  `request_id` String,
  `time` Int64,
  `workspace_id` String,
  `host` String,
  `method` LowCardinality(String),
  `path` String,
  `request_headers` Array(String),
  `request_body` String,
  `response_status` Int32,
  `response_headers` Array(String),
  `response_body` String,
  `error` String,
  `service_latency` Int64,
  `user_agent` String,
  `ip_address` String,
  `country` String,
  `city` String,
  `colo` String,
  `continent` String,
  INDEX `idx_request_id` ((request_id)) TYPE minmax GRANULARITY 1,
  INDEX `idx_workspace_time` ((workspace_id, time)) TYPE minmax GRANULARITY 1
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `time`, `request_id`) ORDER BY (`workspace_id`, `time`, `request_id`) SETTINGS index_granularity = 8192;
-- Create "ratelimits_last_used_v1" table
CREATE TABLE `ratelimits`.`ratelimits_last_used_v1` (
  `time` Int64,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) SETTINGS index_granularity = 8192;
-- Create "ratelimits_per_day_v1" table
CREATE TABLE `ratelimits`.`ratelimits_per_day_v1` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) SETTINGS index_granularity = 8192;
-- Create "ratelimits_per_hour_v1" table
CREATE TABLE `ratelimits`.`ratelimits_per_hour_v1` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) SETTINGS index_granularity = 8192;
-- Create "ratelimits_per_minute_v1" table
CREATE TABLE `ratelimits`.`ratelimits_per_minute_v1` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) SETTINGS index_granularity = 8192;
-- Create "ratelimits_per_month_v1" table
CREATE TABLE `ratelimits`.`ratelimits_per_month_v1` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) SETTINGS index_granularity = 8192;
-- Create "raw_ratelimits_v1" table
CREATE TABLE `ratelimits`.`raw_ratelimits_v1` (
  `request_id` String,
  `time` Int64,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Bool,
  INDEX `idx_request_id` ((request_id)) TYPE minmax GRANULARITY 1,
  INDEX `idx_workspace_time` ((workspace_id, time)) TYPE minmax GRANULARITY 1
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_day_v1" table
CREATE TABLE `verifications`.`key_verifications_per_day_v1` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `time`, `identity_id`, `key_id`) ORDER BY (`workspace_id`, `key_space_id`, `time`, `identity_id`, `key_id`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_day_v2" table
CREATE TABLE `verifications`.`key_verifications_per_day_v2` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`) ORDER BY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_day_v3" table
CREATE TABLE `verifications`.`key_verifications_per_day_v3` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) ORDER BY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_hour_v1" table
CREATE TABLE `verifications`.`key_verifications_per_hour_v1` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `time`, `identity_id`, `key_id`) ORDER BY (`workspace_id`, `key_space_id`, `time`, `identity_id`, `key_id`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_hour_v2" table
CREATE TABLE `verifications`.`key_verifications_per_hour_v2` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`) ORDER BY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_hour_v3" table
CREATE TABLE `verifications`.`key_verifications_per_hour_v3` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) ORDER BY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_minute_v1" table
CREATE TABLE `verifications`.`key_verifications_per_minute_v1` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) ORDER BY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_month_v1" table
CREATE TABLE `verifications`.`key_verifications_per_month_v1` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `time`, `identity_id`, `key_id`) ORDER BY (`workspace_id`, `key_space_id`, `time`, `identity_id`, `key_id`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_month_v2" table
CREATE TABLE `verifications`.`key_verifications_per_month_v2` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`) ORDER BY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_month_v3" table
CREATE TABLE `verifications`.`key_verifications_per_month_v3` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) ORDER BY (`workspace_id`, `key_space_id`, `identity_id`, `key_id`, `time`, `tags`, `outcome`) SETTINGS index_granularity = 8192;
-- Create "raw_key_verifications_v1" table
CREATE TABLE `verifications`.`raw_key_verifications_v1` (
  `request_id` String,
  `time` Int64,
  `workspace_id` String,
  `key_space_id` String,
  `key_id` String,
  `region` LowCardinality(String),
  `outcome` LowCardinality(String),
  `identity_id` String,
  `tags` Array(String) DEFAULT []
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `key_space_id`, `key_id`, `time`) ORDER BY (`workspace_id`, `key_space_id`, `key_id`, `time`) SETTINGS index_granularity = 8192;
-- Create "billable_ratelimits_per_month_mv_v1" view
CREATE MATERIALIZED VIEW `billing`.`billable_ratelimits_per_month_mv_v1` TO `billing`.`billable_ratelimits_per_month_v1` AS SELECT workspace_id, sum(passed) AS count, toYear(time) AS year, toMonth(time) AS month FROM ratelimits.ratelimits_per_month_v1 WHERE passed > 0 GROUP BY workspace_id, year, month;
-- Create "billable_verifications_per_month_mv_v1" view
CREATE MATERIALIZED VIEW `billing`.`billable_verifications_per_month_mv_v1` TO `billing`.`billable_verifications_per_month_v1` AS SELECT workspace_id, count(*) AS count, toYear(time) AS year, toMonth(time) AS month FROM verifications.key_verifications_per_month_v2 WHERE outcome = 'VALID' GROUP BY workspace_id, year, month;
-- Create "billable_verifications_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `billing`.`billable_verifications_per_month_mv_v2` TO `billing`.`billable_verifications_per_month_v2` AS SELECT workspace_id, sum(count) AS count, toYear(time) AS year, toMonth(time) AS month FROM verifications.key_verifications_per_month_v1 WHERE outcome = 'VALID' GROUP BY workspace_id, year, month;
-- Create "active_workspaces_keys_per_month_mv_v1" view
CREATE MATERIALIZED VIEW `business`.`active_workspaces_keys_per_month_mv_v1` TO `business`.`active_workspaces_per_month_v1` AS SELECT workspace_id, toDate(time) AS time FROM verifications.key_verifications_per_month_v2;
-- Create "active_workspaces_ratelimits_per_month_mv_v1" view
CREATE MATERIALIZED VIEW `business`.`active_workspaces_ratelimits_per_month_mv_v1` TO `business`.`active_workspaces_per_month_v1` AS SELECT workspace_id, toDate(time) AS time FROM ratelimits.ratelimits_per_month_v1;
-- Create "api_requests_per_day_mv_v1" view
CREATE MATERIALIZED VIEW `metrics`.`api_requests_per_day_mv_v1` TO `metrics`.`api_requests_per_day_v1` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfDay(fromUnixTimestamp64Milli(time)) AS time FROM metrics.raw_api_requests_v1 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "api_requests_per_hour_mv_v1" view
CREATE MATERIALIZED VIEW `metrics`.`api_requests_per_hour_mv_v1` TO `metrics`.`api_requests_per_hour_v1` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfHour(fromUnixTimestamp64Milli(time)) AS time FROM metrics.raw_api_requests_v1 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "api_requests_per_minute_mv_v1" view
CREATE MATERIALIZED VIEW `metrics`.`api_requests_per_minute_mv_v1` TO `metrics`.`api_requests_per_minute_v1` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM metrics.raw_api_requests_v1 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "ratelimits_last_used_mv_v1" view
CREATE MATERIALIZED VIEW `ratelimits`.`ratelimits_last_used_mv_v1` TO `ratelimits`.`ratelimits_last_used_v1` AS SELECT workspace_id, namespace_id, identifier, maxSimpleState(time) AS time FROM ratelimits.raw_ratelimits_v1 GROUP BY workspace_id, namespace_id, identifier;
-- Create "ratelimits_per_day_mv_v1" view
CREATE MATERIALIZED VIEW `ratelimits`.`ratelimits_per_day_mv_v1` TO `ratelimits`.`ratelimits_per_day_v1` AS SELECT workspace_id, namespace_id, identifier, count(*) AS total, countIf(passed > 0) AS passed, toStartOfDay(fromUnixTimestamp64Milli(time)) AS time FROM ratelimits.raw_ratelimits_v1 GROUP BY workspace_id, namespace_id, identifier, time;
-- Create "ratelimits_per_hour_mv_v1" view
CREATE MATERIALIZED VIEW `ratelimits`.`ratelimits_per_hour_mv_v1` TO `ratelimits`.`ratelimits_per_hour_v1` AS SELECT workspace_id, namespace_id, identifier, countIf(passed > 0) AS passed, count(*) AS total, toStartOfHour(fromUnixTimestamp64Milli(time)) AS time FROM ratelimits.raw_ratelimits_v1 GROUP BY workspace_id, namespace_id, identifier, time;
-- Create "ratelimits_per_minute_mv_v1" view
CREATE MATERIALIZED VIEW `ratelimits`.`ratelimits_per_minute_mv_v1` TO `ratelimits`.`ratelimits_per_minute_v1` AS SELECT workspace_id, namespace_id, identifier, countIf(passed > 0) AS passed, count(*) AS total, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM ratelimits.raw_ratelimits_v1 GROUP BY workspace_id, namespace_id, identifier, time;
-- Create "ratelimits_per_month_mv_v1" view
CREATE MATERIALIZED VIEW `ratelimits`.`ratelimits_per_month_mv_v1` TO `ratelimits`.`ratelimits_per_month_v1` AS SELECT workspace_id, namespace_id, identifier, countIf(passed > 0) AS passed, count(*) AS total, toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time FROM ratelimits.raw_ratelimits_v1 GROUP BY workspace_id, namespace_id, identifier, time;
-- Create "key_verifications_per_day_mv_v1" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_day_mv_v1` TO `verifications`.`key_verifications_per_day_v1` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfDay(fromUnixTimestamp64Milli(time)) AS time FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time;
-- Create "key_verifications_per_day_mv_v2" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_day_mv_v2` TO `verifications`.`key_verifications_per_day_v2` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfDay(fromUnixTimestamp64Milli(time)) AS time, tags FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time, tags;
-- Create "key_verifications_per_day_mv_v3" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_day_mv_v3` TO `verifications`.`key_verifications_per_day_v3` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfDay(fromUnixTimestamp64Milli(time)) AS time, tags FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time, tags;
-- Create "key_verifications_per_hour_mv_v1" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_hour_mv_v1` TO `verifications`.`key_verifications_per_hour_v1` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfHour(fromUnixTimestamp64Milli(time)) AS time FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time;
-- Create "key_verifications_per_hour_mv_v2" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_hour_mv_v2` TO `verifications`.`key_verifications_per_hour_v2` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfHour(fromUnixTimestamp64Milli(time)) AS time, tags FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time, tags;
-- Create "key_verifications_per_hour_mv_v3" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_hour_mv_v3` TO `verifications`.`key_verifications_per_hour_v3` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfHour(fromUnixTimestamp64Milli(time)) AS time, tags FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time, tags;
-- Create "key_verifications_per_minute_mv_v1" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_minute_mv_v1` TO `verifications`.`key_verifications_per_minute_v1` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time, tags FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time, tags;
-- Create "key_verifications_per_month_mv_v1" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_month_mv_v1` TO `verifications`.`key_verifications_per_month_v1` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time;
-- Create "key_verifications_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_month_mv_v2` TO `verifications`.`key_verifications_per_month_v2` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time, tags FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time, tags;
-- Create "key_verifications_per_month_mv_v3" view
CREATE MATERIALIZED VIEW `verifications`.`key_verifications_per_month_mv_v3` TO `verifications`.`key_verifications_per_month_v3` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, count(*) AS count, toStartOfMonth(fromUnixTimestamp64Milli(time)) AS time, tags FROM verifications.raw_key_verifications_v1 GROUP BY workspace_id, key_space_id, identity_id, key_id, outcome, time, tags;
