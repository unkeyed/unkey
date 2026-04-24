-- Audit logs: one row per logical event, with targets as parallel Nested
-- arrays. workspace_id is always the real owner workspace; platform-emitted
-- vs customer-emitted events are distinguished by the `source` column. The
-- bucket column names a logical stream within a workspace (e.g.
-- `unkey_mutations` for platform events, customer-chosen names for
-- customer-emitted events).
--
-- Time fields are stored as Int64 unix-milli for consistency with the rest
-- of Unkey's CH tables (`time` in key_verifications_raw_v2,
-- sentinel_requests_raw_v1, etc.). Partition and TTL expressions wrap them
-- in fromUnixTimestamp64Milli.
--
-- Metadata columns use the native CH JSON type (24.10+). Each JSON column
-- has a `_text` materialized mirror (`toJSONString`) so a single
-- tokenbf_v1 skip index turns "audit logs containing the token X" into a
-- partition-skip query. The dashboard doesn't surface meta search yet but
-- the column is cheap and future-proofs the move.
--
-- Partitioned by inserted_at (not event time) so backfills and late drains
-- land in the current month's part instead of fragmenting old partitions.

CREATE TABLE IF NOT EXISTS default.audit_logs_raw_v1
(
    `event_id`      String,

    -- Event time (millis since epoch). Drives ORDER BY and dashboard time
    -- range filters.
    `time`          Int64 CODEC(Delta, ZSTD(1)),
    -- Wall-clock millis of the CH insert. Drives partitioning so late
    -- events land in the current month's part instead of fragmenting old
    -- ones. The writer always sets this; no DEFAULT.
    `inserted_at`   Int64 CODEC(Delta, ZSTD(1)),

    `workspace_id`  String CODEC(ZSTD(1)),
    `bucket`        LowCardinality(String),
    -- 'platform' for events Unkey emits about a customer's resources;
    -- 'customer' for events emitted via a (future) customer-facing audit
    -- log API. Always 'platform' today.
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
    -- Materialized text mirror of meta so the tokenbf index below works as
    -- a "does the meta contain token X" pre-filter before CH does any JSON
    -- parsing. Empty objects collapse to {} which the index handles as a
    -- no-match cheaply.
    `meta_text`     String MATERIALIZED toJSONString(meta),

    -- Targets stored as parallel arrays. All four arrays MUST be the same
    -- length; the writer is responsible for keeping them aligned.
    `targets.type`  Array(LowCardinality(String)),
    `targets.id`    Array(String),
    `targets.name`  Array(String),
    `targets.meta`  Array(JSON),
    -- Concatenated text mirror across all per-target metas so a single
    -- tokenbf index covers "does ANY target's meta contain X."
    `targets_meta_text` String MATERIALIZED
        arrayStringConcat(arrayMap(x -> toJSONString(x), `targets.meta`), ' '),

    -- Per-row retention stamp, set at insert from the workspace's
    -- retention quota. Stored as unix-milli. Read by the archive cron
    -- (svc/ctrl/worker/auditlogarchive) which exports expired rows to S3
    -- as Parquet, then deletes them via ALTER TABLE DELETE. NOT a CH
    -- TTL clause: the cron is the sole authority on retention so we get
    -- guaranteed compliance-grade archive before deletion.
    `expires_at`    Int64 CODEC(Delta, ZSTD(1)),

    INDEX idx_event             event             TYPE set(100)                GRANULARITY 1,
    INDEX idx_actor_id          actor_id          TYPE bloom_filter(0.01)      GRANULARITY 4,
    INDEX idx_target_id         `targets.id`      TYPE bloom_filter(0.01)      GRANULARITY 4,
    INDEX idx_meta_text         meta_text         TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 4,
    INDEX idx_targets_meta_text targets_meta_text TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 4
)
ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(inserted_at))
ORDER BY (workspace_id, bucket, time, event_id)
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 10000;
