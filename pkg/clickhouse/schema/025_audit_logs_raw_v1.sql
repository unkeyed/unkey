-- Audit logs: one row per (event × target). All rows of a single logical event
-- share the same `event_id`; aggregate with GROUP BY event_id to reconstruct the
-- envelope. `bucket_id` scopes events to a logical stream within a workspace
-- (e.g. `unkey_mutations` for platform events, `aub_xxx` for customer streams).

CREATE TABLE IF NOT EXISTS default.audit_logs_raw_v1
(
    `event_id` String,
    `time` Int64 CODEC(Delta, LZ4),
    `workspace_id` String CODEC(ZSTD(1)),
    `bucket_id` String CODEC(ZSTD(1)),

    `event` String CODEC(ZSTD(1)),
    `description` String CODEC(ZSTD(1)),

    `actor_type` String CODEC(ZSTD(1)),
    `actor_id` String CODEC(ZSTD(1)),
    `actor_name` String CODEC(ZSTD(1)),
    `actor_meta` String CODEC(ZSTD(1)),

    `remote_ip` String,
    `user_agent` String CODEC(ZSTD(1)),
    `meta` String CODEC(ZSTD(1)),

    `target_type` String CODEC(ZSTD(1)),
    `target_id` String CODEC(ZSTD(1)),
    `target_name` String CODEC(ZSTD(1)),
    `target_meta` String CODEC(ZSTD(1)),

    `expires_at` DateTime64(3) DEFAULT now64(3) + INTERVAL 90 DAY,

    INDEX idx_event_id    event_id    TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_actor_id    actor_id    TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_actor_type  actor_type  TYPE bloom_filter(0.01)  GRANULARITY 1,
    INDEX idx_target_id   target_id   TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_target_type target_type TYPE bloom_filter(0.01)  GRANULARITY 1,
    INDEX idx_event       event       TYPE bloom_filter(0.01)  GRANULARITY 1
)
ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(time))
ORDER BY (workspace_id, bucket_id, time, event_id, target_id)
TTL expires_at DELETE
SETTINGS index_granularity = 8192, non_replicated_deduplication_window = 10000;
