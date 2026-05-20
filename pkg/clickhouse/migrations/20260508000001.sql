CREATE TABLE IF NOT EXISTS default.audit_logs_raw_v1
(
    `event_id`      String,

    `time`          Int64 CODEC(Delta, ZSTD(1)),
    `inserted_at`   Int64 CODEC(Delta, ZSTD(1)),

    `workspace_id`  String CODEC(ZSTD(1)),
    `bucket`        LowCardinality(String),
    `source`        LowCardinality(String) DEFAULT 'platform',

    `event`         LowCardinality(String),
    `description`   String CODEC(ZSTD(1)),

    `actor_type`    LowCardinality(String),
    `actor_id`      String CODEC(ZSTD(1)),
    `actor_name`    String CODEC(ZSTD(1)),
    `actor_meta`    JSON   CODEC(ZSTD(1)),

    `remote_ip`     String CODEC(ZSTD(1)),
    `user_agent`    String CODEC(ZSTD(1)),
    `meta`          JSON   CODEC(ZSTD(1)),
    `meta_text`     String MATERIALIZED toJSONString(meta),

    `targets.type`  Array(LowCardinality(String)),
    `targets.id`    Array(String),
    `targets.name`  Array(String),
    `targets.meta`  Array(JSON),
    `targets_meta_text` String MATERIALIZED
        arrayStringConcat(arrayMap(x -> toJSONString(x), `targets.meta`), ' '),

    `correlation_id` String CODEC(ZSTD(1)),

    INDEX idx_event             event             TYPE set(100)                GRANULARITY 1,
    INDEX idx_actor_id          actor_id          TYPE bloom_filter(0.01)      GRANULARITY 4,
    INDEX idx_target_id         `targets.id`      TYPE bloom_filter(0.01)      GRANULARITY 4,
    INDEX idx_meta_text         meta_text         TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 4,
    INDEX idx_targets_meta_text targets_meta_text TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 4,
    INDEX idx_correlation_id    correlation_id    TYPE bloom_filter(0.01)      GRANULARITY 4
)
ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(inserted_at))
ORDER BY (workspace_id, bucket, time, event_id)
-- Flat 90-day retention for ALL workspaces. Per-plan retention is
-- enforced at the dashboard read layer (audit/fetch.ts filters by
-- workspace.quotas.auditLogsRetentionDays); this TTL is the outer
-- bound — "we never keep audit logs longer than 90 days." Bump if
-- the most generous plan grows past 90d.
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 10000;
