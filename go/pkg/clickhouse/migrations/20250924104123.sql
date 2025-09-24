ALTER TABLE `default`.`api_requests_raw_v2` ADD COLUMN `query_string` String;
ALTER TABLE `default`.`api_requests_raw_v2` ADD COLUMN `query_params` Map(String, Array(String));
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `override_id` String;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `limit` UInt64;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `remaining` UInt64;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `reset` Int64 CODEC(Delta(8), LZ4);
-- Drop "temp_sync_metrics_v1_to_v2" view
DROP VIEW `default`.`temp_sync_metrics_v1_to_v2`;
-- Create "temp_sync_metrics_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_metrics_v1_to_v2` TO `default`.`api_requests_raw_v2` AS SELECT request_id, time, workspace_id, host, method, path, '' AS query_string, CAST(mapFromArrays(CAST([], 'Array(String)'), CAST([], 'Array(Array(String))')), 'Map(String, Array(String))') AS query_params, request_headers, request_body, response_status, response_headers, response_body, error, service_latency, user_agent, ip_address, '' AS region FROM metrics.raw_api_requests_v1;
-- Drop "temp_sync_ratelimits_raw_v1_to_v2" view
DROP VIEW `default`.`temp_sync_ratelimits_raw_v1_to_v2`;
-- Create "temp_sync_ratelimits_raw_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_ratelimits_raw_v1_to_v2` TO `default`.`ratelimits_raw_v2` AS SELECT request_id, time, workspace_id, namespace_id, identifier, passed, 0. AS latency, '' AS override_id, 0 AS limit, 0 AS remaining, 0 AS reset FROM ratelimits.raw_ratelimits_v1;
