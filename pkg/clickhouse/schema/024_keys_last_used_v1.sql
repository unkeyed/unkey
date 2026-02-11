-- Materialized view to track the last verification time for each key and identity
-- This dramatically improves query performance for the dashboard's "API Requests" page
--
-- IMPORTANT: Stores ONE row per key (latest verification regardless of outcome).
-- Uses AggregatingMergeTree for automatic aggregation during merges.
-- Can be queried by key_id OR identity_id for flexible last-used tracking.

-- Target table that stores the latest verification per key
CREATE TABLE IF NOT EXISTS `default`.`key_last_used_v1` (
  `workspace_id` String,
  `key_space_id` String,
  `key_id` String,
  `identity_id` String,
  `time` SimpleAggregateFunction(max, Int64),
  `request_id` SimpleAggregateFunction(anyLast, String),
  `outcome` SimpleAggregateFunction(anyLast, LowCardinality(String)),
  `tags` SimpleAggregateFunction(anyLast, Array(String))
) ENGINE = AggregatingMergeTree()
ORDER BY (`workspace_id`, `key_space_id`, `key_id`)
TTL toDateTime(time / 1000) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Materialized view that automatically populates the table from new inserts
CREATE MATERIALIZED VIEW IF NOT EXISTS `default`.`key_last_used_mv_v1`
TO `default`.`key_last_used_v1`
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
GROUP BY workspace_id, key_space_id, key_id;
