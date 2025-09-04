-- Create "api_requests_raw_v2" table
CREATE TABLE `default`.`api_requests_raw_v2` (
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
  `region` LowCardinality(String),
  INDEX `idx_request_id` ((request_id)) TYPE minmax GRANULARITY 1
) ENGINE = MergeTree
PRIMARY KEY (`workspace_id`, `time`, `request_id`) ORDER BY (`workspace_id`, `time`, `request_id`) SETTINGS index_granularity = 8192;
-- Drop "api_requests_per_hour_mv_v1" view
DROP VIEW `default`.`api_requests_per_hour_mv_v1`;
-- Create "api_requests_per_hour_mv_v1" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_hour_mv_v1` TO `default`.`api_requests_per_hour_v1` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfHour(fromUnixTimestamp64Milli(time)) AS time FROM default.api_requests_raw_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Drop "api_requests_per_minute_mv_v1" view
DROP VIEW `default`.`api_requests_per_minute_mv_v1`;
-- Create "api_requests_per_minute_mv_v1" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_minute_mv_v1` TO `default`.`api_requests_per_minute_v1` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time FROM default.api_requests_raw_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Drop "api_requests_per_day_mv_v1" view
DROP VIEW `default`.`api_requests_per_day_mv_v1`;
-- Create "api_requests_per_day_mv_v1" view
CREATE MATERIALIZED VIEW `default`.`api_requests_per_day_mv_v1` TO `default`.`api_requests_per_day_v1` AS SELECT workspace_id, path, response_status, host, method, count(*) AS count, toStartOfDay(fromUnixTimestamp64Milli(time)) AS time FROM default.api_requests_raw_v2 GROUP BY workspace_id, path, response_status, host, method, time;
-- Create "temp_sync_metrics_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_metrics_v1_to_v2` TO `default`.`api_requests_raw_v2` AS SELECT request_id, time, workspace_id, host, method, path, request_headers, request_body, response_status, response_headers, response_body, error, service_latency AS user_agent, ip_address, '' AS region FROM metrics.raw_api_requests_v1;
-- Drop "raw_api_requests_v1" table
DROP TABLE `default`.`raw_api_requests_v1`;
