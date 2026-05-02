-- Runtime logs (v2): customer deployment stdout/stderr captured by the
-- Vector DaemonSet from Krane-managed pods.
--
-- v2 supersedes v1 (023_runtime_logs_raw_v1.sql) to fix the v1 sort-key/
-- cursor mismatch the logdrain hardening pass surfaced.
--
--   v1 sort key: (workspace_id, project_id, environment_id, app_id, time, deployment_id)
--   v2 sort key: (workspace_id, project_id, environment_id, inserted_at, time)
--
-- The logdrain coordinator's cursor query filters on (workspace_id,
-- project_id, environment_id) and orders by (inserted_at, fingerprint).
-- v1's leading three sort columns match the WHERE, but the next column
-- (`app_id`) is not in the WHERE and the order-by column (`inserted_at`)
-- is not in the sort key at all, so ClickHouse re-sorted every matching
-- block by inserted_at on every read. v2 places `inserted_at` directly
-- after the workspace/project/env triple so the cursor scan is a
-- contiguous suffix instead of a per-batch in-block re-sort.
--
-- `app_id` and `deployment_id` move from the sort key into bloom-filter
-- skip indexes — they're filter targets ("show me logs from app X") not
-- ordering columns. Same trade as on the dashboard's substring search
-- columns (`message`, `attributes_text`).
--
-- The cursor uses the inline `cityHash64(deployment_id, time, message)`
-- fingerprint as the (inserted_at, fp) tuple's tiebreaker, so v2 has no
-- stored `row_id` column — nothing in the read path uses one.
CREATE TABLE IF NOT EXISTS default.runtime_logs_raw_v2
(
    -- Application timestamp: when the log line was emitted by the
    -- container, in milliseconds since epoch.
    `time` Int64 CODEC(Delta(8), LZ4),

    -- Ingest timestamp: when the row landed in ClickHouse, in
    -- milliseconds. Drives the logdrain cursor's primary watermark and
    -- the per-row TTL default.
    `inserted_at` Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta(8), LZ4),

    -- Stable per-row id minted by Vector at ingest time as
    -- "log_<16 hex chars>" (~64 bits of entropy). Used as a React key
    -- on the dashboard and as a stable join key for downstream
    -- exporters (logdrain idempotency hint, downstream warehouses).
    -- Not in the sort key or any skip index — no query pattern needs
    -- to filter on it. Empty string for rows backfilled from v1.
    `log_id` String DEFAULT '' CODEC(ZSTD(1)),

    -- Severity is one of `trace|debug|info|warn|error|fatal`. Vector's
    -- VRL transform normalizes the severity field across JSON/logfmt/
    -- plain-text input shapes before insertion.
    `severity` LowCardinality(String) CODEC(ZSTD(1)),
    `message` String CODEC(ZSTD(1)),

    -- Tenant identifiers (from pod labels). The leading triple is the
    -- sort-key prefix the cursor query filters on.
    `workspace_id` String CODEC(ZSTD(1)),
    `project_id` String CODEC(ZSTD(1)),
    `environment_id` String CODEC(ZSTD(1)),
    `app_id` String CODEC(ZSTD(1)),
    `deployment_id` String CODEC(ZSTD(1)),

    -- K8s metadata (which replica produced the line).
    `k8s_pod_name` String CODEC(ZSTD(1)),

    `region` LowCardinality(String) CODEC(ZSTD(1)),
    `platform` LowCardinality(String) CODEC(ZSTD(1)),

    -- Structured payload, parsed from JSON or logfmt by Vector. Empty
    -- object for plain-text logs.
    `attributes` JSON CODEC(ZSTD(1)),

    -- Materialized so the dashboard's substring search can run an
    -- ngrambf-backed positionCaseInsensitive(lower(...)) over the
    -- serialized payload without re-stringifying at query time.
    `attributes_text` String MATERIALIZED toJSONString(attributes) CODEC(ZSTD(1)),

    -- Per-row TTL drives tiered retention (default 90 days from
    -- insert time, configurable per drain in a future iteration).
    `expires_at` DateTime64(3) DEFAULT now64(3) + INTERVAL 90 DAY,

    -- Skip indexes for the filter-y columns that used to live in the
    -- sort key. bloom_filter on the high-cardinality identifiers picks
    -- up "show me logs for this app/deployment" without re-sorting;
    -- the ngrambf_v1 indexes are tuned for the dashboard's substring
    -- search on message + attributes.
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
