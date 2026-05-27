CREATE TABLE build_step_logs_v1
(
  -- When the log line was produced by the build agent. unix milli.
  time Int64 CODEC(Delta(8), ZSTD(1)),

  -- When the row was written to ClickHouse. Drives PARTITION BY so late
  -- flushes from buffered/retried inserts cluster into the part for the
  -- day they arrived, not the day they were produced. Without this,
  -- chatty buffered writers spray rows across many old partitions and
  -- trigger "too many parts" errors.
  --
  -- MATERIALIZED (not DEFAULT) keeps this column server-computed and
  -- excludes it from positional `INSERT INTO table` writes, so the Go
  -- batch writer in pkg/clickhouse/flush.go does not need a matching
  -- struct field.
  inserted_at DateTime64(3) MATERIALIZED now64(3),

  workspace_id String CODEC(ZSTD(1)),
  project_id String CODEC(ZSTD(1)),
  deployment_id String CODEC(ZSTD(1)),
  step_id String CODEC(ZSTD(1)),

  message String CODEC(ZSTD(1)),

  -- Substring search across build logs. Lowercase so
  -- positionCaseInsensitive(lower(message), ...) can use it.
  INDEX idx_message_text_search lower(message) TYPE ngrambf_v1(3, 32768, 2, 0) GRANULARITY 1
)
ENGINE = MergeTree
PARTITION BY toDate(inserted_at)
-- `time` in the sort key lets `ORDER BY time` after the deployment_id
-- filter be a streamed index read (optimize_read_in_order) instead of
-- a full re-sort of the deployment's rows.
ORDER BY (workspace_id, project_id, deployment_id, time, step_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 3 MONTH DELETE
SETTINGS index_granularity = 8192, ttl_only_drop_parts = 1, non_replicated_deduplication_window = 10000
;
