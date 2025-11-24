CREATE TABLE key_verifications_per_minute_v3 (
  time DateTime,
  workspace_id String,
  key_space_id String,
  identity_id String,
  external_id String,
  key_id String,
  outcome LowCardinality (String),
  tags Array(String),
  count SimpleAggregateFunction(sum, Int64),
  spent_credits SimpleAggregateFunction(sum, Int64),
  latency_avg AggregateFunction (avg, Float64),
  latency_p75 AggregateFunction (quantilesTDigest (0.75), Float64),
  latency_p99 AggregateFunction (quantilesTDigest (0.99), Float64),
  INDEX idx_identity_id (identity_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_key_id (key_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_tags (tags) TYPE bloom_filter GRANULARITY 1
) ENGINE = AggregatingMergeTree ()
ORDER BY
  (
    workspace_id,
    time,
    key_space_id,
    identity_id,
    external_id,
    key_id,
    outcome,
    tags
  )
PARTITION BY toStartOfDay(time)
TTL time + INTERVAL 90 DAY DELETE
;

CREATE MATERIALIZED VIEW key_verifications_per_minute_mv_v3 TO key_verifications_per_minute_v3 AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  tags,
  count(*) as count,
  sum(spent_credits) as spent_credits,
  avgState (latency) as latency_avg,
  quantilesTDigestState (0.75) (latency) as latency_p75,
  quantilesTDigestState (0.99) (latency) as latency_p99,
  toStartOfMinute (fromUnixTimestamp64Milli (time)) AS time
FROM
  key_verifications_raw_v2
GROUP BY
  workspace_id,
  time,
  key_space_id,
  identity_id,
  external_id,
  key_id,
  outcome,
  tags
;
