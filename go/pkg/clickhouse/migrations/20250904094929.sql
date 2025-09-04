-- Create "api_requests_per_day_v2" table
CREATE TABLE `default`.`api_requests_per_day_v2` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) ORDER BY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_hour_v2" table
CREATE TABLE `default`.`api_requests_per_hour_v2` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) ORDER BY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) SETTINGS index_granularity = 8192;
-- Create "api_requests_per_minute_v2" table
CREATE TABLE `default`.`api_requests_per_minute_v2` (
  `time` DateTime,
  `workspace_id` String,
  `path` String,
  `response_status` Int32,
  `host` String,
  `method` LowCardinality(String),
  `count` Int64
) ENGINE = SummingMergeTree
PRIMARY KEY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) ORDER BY (`workspace_id`, `time`, `host`, `path`, `response_status`, `method`) SETTINGS index_granularity = 8192;
-- Drop "api_requests_per_day_mv_v1" view
DROP VIEW `default`.`api_requests_per_day_mv_v1`;
-- Drop "api_requests_per_hour_mv_v1" view
DROP VIEW `default`.`api_requests_per_hour_mv_v1`;
-- Drop "api_requests_per_minute_mv_v1" view
DROP VIEW `default`.`api_requests_per_minute_mv_v1`;
-- Drop "temp_sync_metrics_v1_to_v2" view
DROP VIEW `default`.`temp_sync_metrics_v1_to_v2`;
-- Create "temp_sync_metrics_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_metrics_v1_to_v2` TO `default`.`api_requests_raw_v2` AS SELECT request_id, time, workspace_id, host, method, path, request_headers, request_body, response_status, response_headers, response_body, error, service_latency, user_agent, ip_address, '' AS region FROM metrics.raw_api_requests_v1;
-- Create "api_requests_per_day_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_day_mv_v2` TO `default`.`api_requests_per_day_v2` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfDay(fromUnixTimestamp64Milli(time)) AS time FROM default.api_requests_raw_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "api_requests_per_hour_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_hour_mv_v2` TO `default`.`api_requests_per_hour_v2` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfHour(fromUnixTimestamp64Milli(time)) AS time FROM default.api_requests_raw_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "api_requests_per_minute_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_minute_mv_v2` TO `default`.`api_requests_per_minute_v2` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM default.api_requests_raw_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Drop "api_requests_per_day_v1" table
DROP TABLE `default`.`api_requests_per_day_v1`;
-- Drop "api_requests_per_hour_v1" table
DROP TABLE `default`.`api_requests_per_hour_v1`;
-- Drop "api_requests_per_minute_v1" table
DROP TABLE `default`.`api_requests_per_minute_v1`;
