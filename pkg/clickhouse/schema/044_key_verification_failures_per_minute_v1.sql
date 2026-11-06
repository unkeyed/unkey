-- Per-minute rollup of key_verification_failures_raw_v1 for alerts and
-- dashboards that outlive the raw table. count is the sum of sample_weight.
-- key_fingerprints counts distinct submitted keys; fingerprints are scoped to
-- the workspace, so the estimate stays correct within one workspace.

CREATE TABLE IF NOT EXISTS default.key_verification_failures_per_minute_v1
(
    `time`             DateTime,
    `workspace_id`     String,
    `root_key_id`      String,
    `outcome`          LowCardinality(String),
    `reason`           LowCardinality(String),
    `region`           LowCardinality(String),
    `source`           LowCardinality(String),
    `count`            SimpleAggregateFunction(sum, UInt64),
    -- Failures that queried MySQL because the cache had no entry.
    `cache_misses`     SimpleAggregateFunction(sum, UInt64),
    `key_fingerprints` AggregateFunction(uniqCombined64, UInt64),
    `ip_addresses`     AggregateFunction(uniqCombined64, String),
    `key_lengths`      AggregateFunction(topK(5), UInt16),
    `key_prefixes`     AggregateFunction(topK(5), String)
)
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(time)
ORDER BY (workspace_id, time, root_key_id, outcome, reason, region, source)
TTL time + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.key_verification_failures_per_minute_mv_v1
TO default.key_verification_failures_per_minute_v1 AS
SELECT
    toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
    workspace_id,
    root_key_id,
    outcome,
    reason,
    region,
    source,
    toUInt64(sum(sample_weight)) AS count,
    toUInt64(sumIf(sample_weight, cache_result = 'miss')) AS cache_misses,
    uniqCombined64State(key_fingerprint) AS key_fingerprints,
    uniqCombined64State(ip_address) AS ip_addresses,
    topKState(5)(key_length) AS key_lengths,
    topKState(5)(toString(key_prefix)) AS key_prefixes
FROM default.key_verification_failures_raw_v1
GROUP BY
    time,
    workspace_id,
    root_key_id,
    outcome,
    reason,
    region,
    source;
