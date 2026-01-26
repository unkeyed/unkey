-- Create "keys_last_used_v2" table
CREATE TABLE IF NOT EXISTS `default`.`keys_last_used_v2` (
  `workspace_id` String,
  `key_space_id` String,
  `key_id` String,
  `time` Int64 CODEC(Delta, LZ4),
  `request_id` String,
  `outcome` LowCardinality(String),
  `tags` Array(String) DEFAULT []
) ENGINE = ReplacingMergeTree(time)
ORDER BY (`workspace_id`, `key_space_id`, `key_id`, `time`) TTL toDateTime(fromUnixTimestamp64Milli(time)) + toIntervalDay(90);
-- Create "keys_last_used_mv_v2" view
CREATE MATERIALIZED VIEW IF NOT EXISTS `default`.`keys_last_used_mv_v2` TO `default`.`keys_last_used_v2` AS SELECT workspace_id, key_space_id, key_id, time, request_id, outcome, tags FROM default.key_verifications_raw_v2;
