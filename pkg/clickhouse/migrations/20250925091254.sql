ALTER TABLE `default`.`api_requests_raw_v2` ADD COLUMN `query_string` String;
ALTER TABLE `default`.`api_requests_raw_v2` ADD COLUMN `query_params` Map(String, Array(String));
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `override_id` String;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `limit` UInt64;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `remaining` UInt64;
ALTER TABLE `default`.`ratelimits_raw_v2` ADD COLUMN `reset_at` Int64 CODEC(Delta(8), LZ4);
