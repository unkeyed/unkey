-- Materialized view to track the last verification time for each key
-- This dramatically improves query performance for the dashboard's "API Requests" page
-- by allowing queries to scan ~10K rows instead of 150M+ rows
--
-- IMPORTANT: Stores ONE row per key (latest verification regardless of outcome).
-- ORDER BY prioritizes time for efficient "most recently used keys" queries.

-- Target table that stores the latest verification per key
CREATE TABLE IF NOT EXISTS default.keys_last_used_v2
(
    workspace_id String,
    key_space_id String,
    time Int64 CODEC(Delta, LZ4),
    key_id String,
    request_id String,
    outcome LowCardinality(String),
    tags Array(String) DEFAULT []
)
ENGINE = ReplacingMergeTree(time)
ORDER BY (workspace_id, key_space_id, time, key_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE;

-- Materialized view that automatically populates the table from new inserts
CREATE MATERIALIZED VIEW IF NOT EXISTS default.keys_last_used_mv_v2
TO default.keys_last_used_v2
AS
SELECT
    workspace_id,
    key_space_id,
    key_id,
    time,
    request_id,
    outcome,
    tags
FROM default.key_verifications_raw_v2;
