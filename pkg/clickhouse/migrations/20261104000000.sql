-- Key verification failures: one row for every verifyKey call that does not
-- return VALID, including the early NOT_FOUND returns that never reach
-- key_verifications_raw_v2. Internal only: no customer analytics, billing, or
-- log drain reads this table, so it may hold reasons and key IDs callers must
-- not see. It never holds the raw key or its sha256 hash.
--
-- Above a per-pod cap, svc/api keeps one row in 100 with sample_weight = 100.
-- Count failures with sum(sample_weight), not count().

CREATE TABLE `default`.`key_verification_failures_raw_v1`
(
    `request_id`       String CODEC(ZSTD(1)),
    `time`             Int64 CODEC(Delta, ZSTD(1)),
    -- The caller's authorized workspace, not the key owner's.
    `workspace_id`     String CODEC(ZSTD(1)),
    `principal_type`   LowCardinality(String),
    -- Empty unless a root key made the call.
    `root_key_id`      String CODEC(ZSTD(1)),
    `region`           LowCardinality(String),
    `source`           LowCardinality(String),
    `ip_address`       String CODEC(ZSTD(1)),
    `user_agent`       String CODEC(ZSTD(1)),
    -- The code the caller received.
    `outcome`          LowCardinality(String),
    -- The internal cause, for example unknown_hash or workspace_mismatch.
    `reason`           LowCardinality(String),
    -- 'hit', 'null' (cached unknown hash), 'miss' (unknown hash that queried
    -- MySQL), or empty when no cache lookup ran.
    `cache_result`     LowCardinality(String),
    -- Set when the hash matched a key, including a key in another workspace.
    `key_id`           String CODEC(ZSTD(1)),
    `key_space_id`     String CODEC(ZSTD(1)),
    `key_workspace_id` String CODEC(ZSTD(1)),
    `key_length`       UInt16,
    -- Text before the first underscore, only when it is 1 to 8 letters or
    -- digits and the key is at least 16 bytes long.
    `key_prefix`       LowCardinality(String),
    -- hex, base58, alnum, base64url, other, or empty.
    `key_charset`      LowCardinality(String),
    `sample_weight`    UInt32 DEFAULT 1,

    INDEX idx_request_id request_id TYPE bloom_filter GRANULARITY 1,
    INDEX idx_root_key_id root_key_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_key_id key_id TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(fromUnixTimestamp64Milli(time))
ORDER BY (workspace_id, time, reason)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 30 DAY DELETE
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 10000,
         ttl_only_drop_parts = 1;

-- Per-minute rollup of key_verification_failures_raw_v1 for alerts and
-- dashboards that outlive the raw table. count is the sum of sample_weight.

CREATE TABLE `default`.`key_verification_failures_per_minute_v1`
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
    `ip_addresses`     AggregateFunction(uniqCombined64, String),
    `key_lengths`      AggregateFunction(topK(5), UInt16),
    `key_prefixes`     AggregateFunction(topK(5), String)
)
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(time)
ORDER BY (workspace_id, time, root_key_id, outcome, reason, region, source)
TTL time + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW `default`.`key_verification_failures_per_minute_mv_v1`
TO `default`.`key_verification_failures_per_minute_v1` AS
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
