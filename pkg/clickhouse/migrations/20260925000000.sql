-- Staff back office audit log: one row per staff edit. The back office
-- writes rows by draining the MySQL backoffice_audit_outbox table and reads
-- them for its own audit views. No service in this repo uses the table.
--
-- Lives in its own database, apart from default.audit_logs_raw_v1, so staff
-- actions never show in a customer's audit log and per-workspace ClickHouse
-- users, whose grants cover only default tables, cannot read it.
--
-- ReplacingMergeTree on (time, event_id) collapses events the drainer sends
-- again after a failed attempt.

CREATE DATABASE IF NOT EXISTS `backoffice`;

CREATE TABLE `backoffice`.`audit_logs_v1`
(
    -- boal_ id; the dedupe key.
    `event_id`     String,
    -- Unix milliseconds.
    `time`         Int64 CODEC(Delta, ZSTD(1)),

    -- WorkOS user id of the staff member. Name and email are not stored;
    -- look them up in WorkOS by id.
    `actor_id`     String CODEC(ZSTD(1)),

    `table_name`   LowCardinality(String),
    -- 'insert' | 'update' | 'softDelete' | 'restore'.
    `action`       LowCardinality(String),
    `pk_value`     String CODEC(ZSTD(1)),
    -- Empty when the edited table has no workspace link.
    `workspace_id` String CODEC(ZSTD(1)),
    -- JSON [{field, before, after}], changed fields only.
    `changes`      String CODEC(ZSTD(1)),
    `reason`       String CODEC(ZSTD(1)),
    `request_ip`   String CODEC(ZSTD(1)),
    `user_agent`   String CODEC(ZSTD(1)),

    INDEX idx_actor_id     actor_id     TYPE bloom_filter GRANULARITY 1,
    INDEX idx_workspace_id workspace_id TYPE bloom_filter GRANULARITY 1,
    INDEX idx_pk_value     pk_value     TYPE bloom_filter GRANULARITY 1
)
ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMM(fromUnixTimestamp64Milli(time))
ORDER BY (time, event_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE
SETTINGS index_granularity = 8192,
         non_replicated_deduplication_window = 10000;
