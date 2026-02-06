-- Create "keys_last_used_v1" table with AggregatingMergeTree for pre-aggregated data
CREATE TABLE IF NOT EXISTS `default`.`keys_last_used_v1` (
  `workspace_id` String,
  `key_space_id` String,
  `key_id` String,
  `identity_id` String,
  `time` SimpleAggregateFunction(max, Int64),
  `request_id` SimpleAggregateFunction(anyLast, String),
  `outcome` SimpleAggregateFunction(anyLast, LowCardinality(String)),
  `tags` SimpleAggregateFunction(anyLast, Array(String))
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `key_space_id`, `key_id`, `time`)
TTL toDateTime(time / 1000) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Create "keys_last_used_mv_v1" materialized view that pre-aggregates per key
CREATE MATERIALIZED VIEW IF NOT EXISTS `default`.`keys_last_used_mv_v1`
TO `default`.`keys_last_used_v1`
AS
SELECT
  workspace_id,
  key_space_id,
  key_id,
  identity_id,
  max(time) as time,
  anyLast(request_id) as request_id,
  anyLast(outcome) as outcome,
  anyLast(tags) as tags
FROM `default`.`key_verifications_raw_v2`
GROUP BY workspace_id, key_space_id, key_id, identity_id;
