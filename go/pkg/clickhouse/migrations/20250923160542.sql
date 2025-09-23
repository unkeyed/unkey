ALTER TABLE `default`.`active_workspaces_per_month_v2` MODIFY PARTITION BY (toYYYYMM(time));
ALTER TABLE `default`.`api_requests_per_day_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`api_requests_per_hour_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`api_requests_per_minute_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`api_requests_per_month_v2` MODIFY PARTITION BY (toYYYYMM(time));
ALTER TABLE `default`.`api_requests_raw_v2` MODIFY PARTITION BY (toYYYYMMDD(fromUnixTimestamp64Milli(time)));
ALTER TABLE `default`.`billable_ratelimits_per_month_v2` MODIFY PARTITION BY ((year, month));
ALTER TABLE `default`.`billable_verifications_per_month_v2` MODIFY PARTITION BY ((year, month));
ALTER TABLE `default`.`key_verifications_per_day_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`key_verifications_per_hour_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`key_verifications_per_minute_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`key_verifications_per_month_v2` MODIFY PARTITION BY (toYYYYMM(time));
ALTER TABLE `default`.`key_verifications_raw_v2` MODIFY PARTITION BY (toYYYYMMDD(fromUnixTimestamp64Milli(time)));
ALTER TABLE `default`.`ratelimits_per_day_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`ratelimits_per_hour_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`ratelimits_per_minute_v2` MODIFY PARTITION BY (toYYYYMMDD(time));
ALTER TABLE `default`.`ratelimits_per_month_v2` MODIFY PARTITION BY (toYYYYMM(time));
ALTER TABLE `default`.`ratelimits_raw_v2` MODIFY PARTITION BY (toYYYYMMDD(fromUnixTimestamp64Milli(time)));
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `override_id` String;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `limit` UInt64;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `remaining` UInt64;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `reset` Int64 CODEC(Delta(8), LZ4);
-- Drop "temp_sync_ratelimits_raw_v1_to_v2" view
DROP VIEW `default`.`temp_sync_ratelimits_raw_v1_to_v2`;
-- Create "temp_sync_ratelimits_raw_v1_to_v2" view
CREATE MATERIALIZED VIEW `default`.`temp_sync_ratelimits_raw_v1_to_v2` TO `default`.`ratelimits_raw_v2` AS SELECT request_id, time, workspace_id, namespace_id, identifier, passed, 0. AS latency, '' AS override_id, 0 AS limit, 0 AS remaining, 0 AS reset FROM ratelimits.raw_ratelimits_v1;
