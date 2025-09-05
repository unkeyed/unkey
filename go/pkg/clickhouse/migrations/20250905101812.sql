-- Create "active_workspaces_per_month_v2" table
CREATE TABLE `default`.`active_workspaces_per_month_v2` (
  `time` Date,
  `workspace_id` String
) ENGINE = ReplacingMergeTree
PRIMARY KEY (`time`, `workspace_id`) ORDER BY (`time`, `workspace_id`) TTL time + toIntervalYear(5) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_day_v2" table
CREATE TABLE `default`.`api_requests_per_day_v2` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64,
  INDEX `idx_host` ((host)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_method` ((method)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_path` ((path)) TYPE bloom_filter GRANULARITY 1
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `response_status`, `host`, `method`, `path`) ORDER BY (`workspace_id`, `time`, `response_status`, `host`, `method`, `path`) TTL time + toIntervalDay(100) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_hour_v2" table
CREATE TABLE `default`.`api_requests_per_hour_v2` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64,
  INDEX `idx_host` ((host)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_method` ((method)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_path` ((path)) TYPE bloom_filter GRANULARITY 1
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `response_status`, `host`, `method`, `path`) ORDER BY (`workspace_id`, `time`, `response_status`, `host`, `method`, `path`) TTL time + toIntervalDay(30) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_minute_v2" table
CREATE TABLE `default`.`api_requests_per_minute_v2` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64,
  INDEX `idx_host` ((host)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_method` ((method)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_path` ((path)) TYPE bloom_filter GRANULARITY 1
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `response_status`, `host`, `method`, `path`) ORDER BY (`workspace_id`, `time`, `response_status`, `host`, `method`, `path`) TTL time + toIntervalDay(7) SETTINGS index_granularity = 8192;
-- Create "api_requests_raw_v2" table
CREATE TABLE `default`.`api_requests_raw_v2` (
  `request_id` String,
  `time` Int64 CODEC(Delta(8), LZ4),
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
  `region` LowCardinality(String),
  INDEX `idx_request_id` ((request_id)) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `time`, `request_id`) ORDER BY (`workspace_id`, `time`, `request_id`) TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalMonth(1) SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
-- Create "billable_ratelimits_per_month_v2" table
CREATE TABLE `default`.`billable_ratelimits_per_month_v2` (
  `year` Int16,
  `month` Int8,
  `workspace_id` String,
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `year`, `month`) ORDER BY (`workspace_id`, `year`, `month`) SETTINGS index_granularity = 8192;
-- Create "billable_verifications_per_month_v2" table
CREATE TABLE `default`.`billable_verifications_per_month_v2` (
  `year` Int16,
  `month` Int8,
  `workspace_id` String,
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `year`, `month`) ORDER BY (`workspace_id`, `year`, `month`) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_day_v2" table
CREATE TABLE `default`.`key_verifications_per_day_v2` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64,
  `spent_credits` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) TTL time + toIntervalDay(100) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_hour_v2" table
CREATE TABLE `default`.`key_verifications_per_hour_v2` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64,
  `spent_credits` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) TTL time + toIntervalDay(30) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_minute_v2" table
CREATE TABLE `default`.`key_verifications_per_minute_v2` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64,
  `spent_credits` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) TTL time + toIntervalDay(7) SETTINGS index_granularity = 8192;
-- Create "key_verifications_per_month_v2" table
CREATE TABLE `default`.`key_verifications_per_month_v2` (
  `time` DateTime,
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String),
  `count` Int64,
  `spent_credits` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `identity_id`, `key_id`, `outcome`, `tags`) TTL time + toIntervalYear(3) SETTINGS index_granularity = 8192;
-- Create "key_verifications_raw_v2" table
CREATE TABLE `default`.`key_verifications_raw_v2` (
  `request_id` String,
  `time` Int64 CODEC(Delta(8), LZ4),
  `workspace_id` String,
  `key_space_id` String,
  `identity_id` String,
  `key_id` String,
  `region` LowCardinality(String),
  `outcome` LowCardinality(String),
  `tags` Array(String) DEFAULT [],
  `spent_credits` Int64,
  `latency` Float64,
  INDEX `idx_identity_id` ((identity_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_key_id` ((key_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_request_id` ((request_id)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_tags` ((tags)) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `time`, `key_space_id`, `outcome`) ORDER BY (`workspace_id`, `time`, `key_space_id`, `outcome`) TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalMonth(1) SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
-- Create "ratelimits_per_day_v2" table
CREATE TABLE `default`.`ratelimits_per_day_v2` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identifier` ((identifier)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) TTL time + toIntervalDay(100) SETTINGS index_granularity = 8192;
-- Create "ratelimits_per_hour_v2" table
CREATE TABLE `default`.`ratelimits_per_hour_v2` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identifier` ((identifier)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) TTL time + toIntervalDay(30) SETTINGS index_granularity = 8192;
-- Create "ratelimits_per_minute_v2" table
CREATE TABLE `default`.`ratelimits_per_minute_v2` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identifier` ((identifier)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) TTL time + toIntervalDay(7) SETTINGS index_granularity = 8192;
-- Create "ratelimits_per_month_v2" table
CREATE TABLE `default`.`ratelimits_per_month_v2` (
  `time` DateTime,
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Int64,
  `total` Int64,
  `latency_avg` AggregateFunction(avg, Float64),
  `latency_p75` AggregateFunction(quantilesTDigest(0.75), Float64),
  `latency_p99` AggregateFunction(quantilesTDigest(0.99), Float64),
  INDEX `idx_identifier` ((identifier)) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree
PRIMARY KEY (`workspace_id`, `namespace_id`, `time`, `identifier`) ORDER BY (`workspace_id`, `namespace_id`, `time`, `identifier`) TTL time + toIntervalYear(3) SETTINGS index_granularity = 8192;
-- Create "ratelimits_raw_v2" table
CREATE TABLE `default`.`ratelimits_raw_v2` (
  `request_id` String,
  `time` Int64 CODEC(Delta(8), LZ4),
  `workspace_id` String,
  `namespace_id` String,
  `identifier` String,
  `passed` Bool,
  `latency` Float64,
  INDEX `idx_identifier` ((identifier)) TYPE bloom_filter GRANULARITY 1,
  INDEX `idx_request_id` ((request_id)) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `time`, `namespace_id`) ORDER BY (`workspace_id`, `time`, `namespace_id`) TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalMonth(1) SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
-- Create "active_workspaces_keys_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`active_workspaces_keys_per_month_mv_v2` TO `default`.`active_workspaces_per_month_v2` AS SELECT workspace_id, toDate(time) AS time FROM default.key_verifications_per_month_v2;
-- Create "active_workspaces_ratelimits_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`active_workspaces_ratelimits_per_month_mv_v2` TO `default`.`active_workspaces_per_month_v2` AS SELECT workspace_id, toDate(time) AS time FROM default.ratelimits_per_month_v2;
-- Create "api_requests_per_day_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_day_mv_v2` TO `default`.`api_requests_per_day_v2` AS SELECT workspace_id, path, response_status, host, method, sum(count) AS count, toStartOfDay(time) AS time FROM default.api_requests_per_hour_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "api_requests_per_hour_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_hour_mv_v2` TO `default`.`api_requests_per_hour_v2` AS SELECT workspace_id, path, response_status, host, method, sum(count) AS count, toStartOfHour(time) AS time FROM default.api_requests_per_minute_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "api_requests_per_minute_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_minute_mv_v2` TO `default`.`api_requests_per_minute_v2` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM default.api_requests_raw_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "billable_ratelimits_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`billable_ratelimits_per_month_mv_v2` TO `default`.`billable_ratelimits_per_month_v2` AS SELECT workspace_id, sum(passed) AS count, toYear(time) AS year, toMonth(time) AS month FROM default.ratelimits_per_month_v2 GROUP BY workspace_id, year, month;
-- Create "billable_verifications_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`billable_verifications_per_month_mv_v2` TO `default`.`billable_verifications_per_month_v2` AS SELECT workspace_id, sum(count) AS count, toYear(time) AS year, toMonth(time) AS month FROM default.key_verifications_per_month_v2 WHERE outcome = 'VALID' GROUP BY workspace_id, year, month;
-- Create "key_verifications_per_day_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_day_mv_v2` TO `default`.`key_verifications_per_day_v2` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfDay(time) AS time FROM default.key_verifications_per_hour_v2 GROUP BY workspace_id, time, key_space_id, identity_id, key_id, outcome, tags;
-- Create "key_verifications_per_hour_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_hour_mv_v2` TO `default`.`key_verifications_per_hour_v2` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfHour(time) AS time FROM default.key_verifications_per_minute_v2 GROUP BY workspace_id, time, key_space_id, identity_id, key_id, outcome, tags;
-- Create "key_verifications_per_minute_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_minute_mv_v2` TO `default`.`key_verifications_per_minute_v2` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, tags, count(*) AS count, sum(spent_credits) AS spent_credits, avgState(latency) AS latency_avg, quantilesTDigestState(0.75)(latency) AS latency_p75, quantilesTDigestState(0.99)(latency) AS latency_p99, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM default.key_verifications_raw_v2 GROUP BY workspace_id, time, key_space_id, identity_id, key_id, outcome, tags;
-- Create "key_verifications_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`key_verifications_per_month_mv_v2` TO `default`.`key_verifications_per_month_v2` AS SELECT workspace_id, key_space_id, identity_id, key_id, outcome, tags, sum(count) AS count, sum(spent_credits) AS spent_credits, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfMonth(time) AS time FROM default.key_verifications_per_day_v2 GROUP BY workspace_id, time, key_space_id, identity_id, key_id, outcome, tags;
-- Create "ratelimits_per_day_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_day_mv_v2` TO `default`.`ratelimits_per_day_v2` AS SELECT workspace_id, namespace_id, identifier, sum(total) AS total, sum(passed) AS passed, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfDay(time) AS time FROM default.ratelimits_per_hour_v2 GROUP BY workspace_id, namespace_id, time, identifier;
-- Create "ratelimits_per_hour_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_hour_mv_v2` TO `default`.`ratelimits_per_hour_v2` AS SELECT workspace_id, namespace_id, identifier, sum(total) AS total, sum(passed) AS passed, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfHour(time) AS time FROM default.ratelimits_per_minute_v2 GROUP BY workspace_id, namespace_id, time, identifier;
-- Create "ratelimits_per_minute_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_minute_mv_v2` TO `default`.`ratelimits_per_minute_v2` AS SELECT workspace_id, namespace_id, identifier, count(*) AS total, countIf(passed > 0) AS passed, avgState(latency) AS latency_avg, quantilesTDigestState(0.75)(latency) AS latency_p75, quantilesTDigestState(0.99)(latency) AS latency_p99, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM default.ratelimits_raw_v2 GROUP BY workspace_id, namespace_id, time, identifier;
-- Create "ratelimits_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`ratelimits_per_month_mv_v2` TO `default`.`ratelimits_per_month_v2` AS SELECT workspace_id, namespace_id, identifier, sum(total) AS total, sum(passed) AS passed, avgMergeState(latency_avg) AS latency_avg, quantilesTDigestMergeState(0.75)(latency_p75) AS latency_p75, quantilesTDigestMergeState(0.99)(latency_p99) AS latency_p99, toStartOfMonth(time) AS time FROM default.ratelimits_per_day_v2 GROUP BY workspace_id, namespace_id, time, identifier;
-- Create "temp_sync_key_verifications_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_key_verifications_v1_to_v2` TO `default`.`key_verifications_raw_v2` AS SELECT request_id, time, workspace_id, key_space_id, identity_id, key_id, region, outcome, tags, 0 AS spent_credits, 0. AS latency FROM verifications.raw_key_verifications_v1;
-- Create "temp_sync_metrics_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_metrics_v1_to_v2` TO `default`.`api_requests_raw_v2` AS SELECT request_id, time, workspace_id, host, method, path, request_headers, request_body, response_status, response_headers, response_body, error, service_latency, user_agent, ip_address, '' AS region FROM metrics.raw_api_requests_v1;
-- Create "temp_sync_ratelimits_raw_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_ratelimits_raw_v1_to_v2` TO `default`.`ratelimits_raw_v2` AS SELECT request_id, time, workspace_id, namespace_id, identifier, passed, 0. AS latency FROM ratelimits.raw_ratelimits_v1;
