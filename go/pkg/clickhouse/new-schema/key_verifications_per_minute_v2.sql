CREATE TABLE key_verifications_per_minute_v2
(
  time          DateTime,
  workspace_id  String,
  key_space_id  String,
  identity_id   String,
  key_id        String,
  outcome       LowCardinality(String),
  tags          Array(String),
  count         Int64
  latency_avg   AggregateFunction(avg, Float64),
  latency_p75   AggregateFunction(quantilesTDigest(0.75), Float64),
  latency_p99   AggregateFunction(quantilesTDigest(0.99), Float64)
)
ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(time)
ORDER BY (workspace_id, time, key_space_id, identity_id, key_id, tags, outcome)
TTL time + INTERVAL 7 DAY DELETE
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW key_verifications_per_minute_mv_v2
TO key_verifications_per_minute_v2
AS
SELECT
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  count(*) as count,
  avgState(latency) as latency_avg,
  quantilesTDigestState(0.75)(latency) as latency_p75,
  quantilesTDigestState(0.99)(latency) as latency_p99,
  toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time
FROM key_verifications_raw_v2
GROUP BY
  workspace_id,
  key_space_id,
  identity_id,
  key_id,
  outcome,
  time
;
