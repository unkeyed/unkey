-- Materialized view to track the last verification time for each key
-- This dramatically improves query performance for the dashboard's "API Requests" page
-- by allowing queries to scan ~10K rows instead of 150M+ rows
--
-- IMPORTANT: Stores ONE row per key (latest verification regardless of outcome).
-- Uses AggregatingMergeTree for automatic aggregation during merges.

-- Target table that stores the latest verification per key
CREATE TABLE IF NOT EXISTS default.keys_last_used_v2
(
    workspace_id String,
    key_space_id String,
    key_id String,
    time SimpleAggregateFunction(max, Int64),
    request_id SimpleAggregateFunction(anyLast, String),
    outcome SimpleAggregateFunction(anyLast, LowCardinality(String)),
    tags SimpleAggregateFunction(anyLast, Array(String))
)
ENGINE = AggregatingMergeTree()
PRIMARY KEY (workspace_id, key_space_id, key_id)
ORDER BY (workspace_id, key_space_id, key_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE;

-- Materialized view that automatically populates the table from new inserts
CREATE MATERIALIZED VIEW IF NOT EXISTS default.keys_last_used_mv_v2
TO default.keys_last_used_v2
AS
SELECT
    workspace_id,
    key_space_id,
    key_id,
    max(time) as time,
    anyLast(request_id) as request_id,
    anyLast(outcome) as outcome,
    anyLast(tags) as tags
FROM default.key_verifications_raw_v2
GROUP BY workspace_id, key_space_id, key_id;

