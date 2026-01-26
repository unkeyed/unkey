-- Create "keys_last_used_v2" table
CREATE TABLE `default`.`keys_last_used_v2` (
  `workspace_id` String,
  `key_space_id` String,
  `time` Int64 CODEC(Delta(8), LZ4),
  `key_id` String,
  `request_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String) DEFAULT []
) ENGINE = ReplacingMergeTree(time)
PRIMARY KEY (`workspace_id`, `key_space_id`, `time`, `key_id`) ORDER BY (`workspace_id`, `key_space_id`, `time`, `key_id`) TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(90) SETTINGS index_granularity = 8192;
-- Create "keys_last_used_mv_v2" materialized view
CREATE MATERIALIZED VIEW `default`.`keys_last_used_mv_v2` TO `default`.`keys_last_used_v2` AS
SELECT
  workspace_id,
  key_space_id,
  key_id,
  time,
  request_id,
  outcome,
  tags
FROM `default`.`key_verifications_raw_v2`;
