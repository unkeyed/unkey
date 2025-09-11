CREATE TABLE key_verifications_per_month_v2 (
  time Date,
  workspace_id String,
  key_space_id String,
  identity_id String,
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
PARTITION BY
  toYYYYMM (time)
ORDER BY
  (
    workspace_id,
    time,
    key_space_id,
    identity_id,
    key_id,
    outcome,
    tags
  )
TTL time + INTERVAL 3 YEAR DELETE
;

CREATE MATERIALIZED VIEW key_verifications_per_month_mv_v2 TO key_verifications_per_month_v2 AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  tags,
  sum(count) as count,
  sum(spent_credits) as spent_credits,
  avgMergeState(latency_avg) as latency_avg,
  quantilesTDigestMergeState(0.75)(latency_p75) as latency_p75,
  quantilesTDigestMergeState(0.99)(latency_p99) as latency_p99,
  toDate(toStartOfMonth(time)) AS time
FROM
  key_verifications_per_day_v2
GROUP BY
  workspace_id,
  time,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  tags
;
