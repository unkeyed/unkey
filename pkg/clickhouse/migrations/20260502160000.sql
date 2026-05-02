-- Create runtime_logs_raw_v2 to fix the v1 sort-key/cursor mismatch.
--
-- v1 sort key: (workspace_id, project_id, environment_id, app_id, time, deployment_id)
-- The cursor query filters on (workspace_id, project_id, environment_id) and
-- orders by (inserted_at, fingerprint). v1's leading three sort columns
-- match the WHERE, but the next column (`app_id`) is not in the WHERE and
-- the order-by column (`inserted_at`) is not in the sort key at all, so
-- ClickHouse re-sorts every matching block by inserted_at — exactly the
-- pathology hardening-plan item #10 flagged.
--
-- v2 sort key: (workspace_id, project_id, environment_id, inserted_at, time)
-- The cursor query is now fully sort-key aligned: leading three columns
-- match the WHERE; `inserted_at` is the next column, so the (inserted_at,
-- fingerprint) > (cursor.timeMs, cursor.rowID) bound prunes the read to
-- a contiguous suffix of the matching block instead of forcing a full
-- in-block sort. `time` is the final tiebreaker — it isn't strictly
-- unique, but the application-side `cityHash64(deployment_id, time,
-- message)` fingerprint already provides total order at the read level,
-- so we don't need a synthetic uniqueness column in the sort key.
--
-- `app_id` and `deployment_id` move from the sort key into bloom-filter
-- skip indexes — they're filter targets ("show me logs from app X") not
-- ordering columns. Same trade as on the dashboard's substring search
-- columns (`message`, `attributes_text`): keep them out of the sort key,
-- pay for them with skip indexes only when a query touches them.
--
-- The cursor uses the inline `cityHash64(deployment_id, time, message)`
-- fingerprint as the (time, fp) tuple's tiebreaker, so v2 has no stored
-- `row_id` column — there is no use of one anywhere in the read path.

CREATE TABLE IF NOT EXISTS default.runtime_logs_raw_v2
(
    `time` Int64 CODEC(Delta(8), LZ4),
    `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta(8), LZ4),

    -- Stable per-row id minted by Vector at ingest time as
    -- "log_<16 hex chars>" (~64 bits of entropy). Used as a React key on
    -- the dashboard and as a stable join key for downstream exporters
    -- (logdrain idempotency hint, downstream warehouses). Not in the
    -- sort key or any skip index — no query pattern needs to filter on
    -- it. Empty string for the rows backfilled from v1.
    `log_id` String DEFAULT '' CODEC(ZSTD(1)),

    `severity` LowCardinality(String) CODEC(ZSTD(1)),
    `message` String CODEC(ZSTD(1)),

    `workspace_id` String CODEC(ZSTD(1)),
    `project_id` String CODEC(ZSTD(1)),
    `environment_id` String CODEC(ZSTD(1)),
    `app_id` String CODEC(ZSTD(1)),
    `deployment_id` String CODEC(ZSTD(1)),

    `k8s_pod_name` String CODEC(ZSTD(1)),

    `region` LowCardinality(String) CODEC(ZSTD(1)),
    `platform` LowCardinality(String) CODEC(ZSTD(1)),

    `attributes` JSON CODEC(ZSTD(1)),
    `attributes_text` String MATERIALIZED toJSONString(attributes) CODEC(ZSTD(1)),

    `expires_at` DateTime64(3) DEFAULT now64(3) + INTERVAL 90 DAY,

    -- Skip indexes for the filter-y columns that used to live in the
    -- sort key. bloom_filter on the high-cardinality identifiers picks
    -- up "show me logs for this app/deployment" without re-sorting; the
    -- ngrambf_v1 indexes are tuned for the dashboard's substring search
    -- on message + attributes (same pattern as trigger.dev v2).
    INDEX idx_app_id app_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_deployment_id deployment_id TYPE bloom_filter(0.001) GRANULARITY 1,
    INDEX idx_message_search lower(message) TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1,
    INDEX idx_attributes_search lower(attributes_text) TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1,
    INDEX idx_time time TYPE minmax GRANULARITY 1
)
ENGINE = MergeTree()
PARTITION BY toDate(fromUnixTimestamp64Milli(inserted_at))
ORDER BY (workspace_id, project_id, environment_id, inserted_at, time)
TTL toDateTime(expires_at) + INTERVAL 7 DAY
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1;

-- Backfill: copy every existing v1 row into v2. Idempotent because the
-- migration runs once per Atlas timestamp; if it crashes mid-INSERT, the
-- next run sees the partial v2 state and re-running this exact INSERT
-- would double-count. For the cutover we accept that risk and rely on
-- the operator running the migration in a single shot — at production
-- scale the alternative (a CH-side dedup MV) is overkill for a one-off
-- v1→v2 lift. Drop runtime_logs_raw_v1 once Vector and logdrain are
-- both pointing at v2 and the dashboards have been migrated.
-- log_id is omitted from the column list so the table's `DEFAULT ''`
-- applies to every backfilled row. v1 didn't carry the column, so every
-- pre-cutover row gets the empty-string sentinel; only post-cutover
-- rows minted by Vector carry the real "log_<16 hex chars>" id.
INSERT INTO default.runtime_logs_raw_v2
    (time, inserted_at, severity, message,
     workspace_id, project_id, environment_id, app_id, deployment_id,
     k8s_pod_name, region, platform, attributes, expires_at)
SELECT time, inserted_at, severity, message,
       workspace_id, project_id, environment_id, app_id, deployment_id,
       k8s_pod_name, region, platform, attributes, expires_at
FROM default.runtime_logs_raw_v1;
