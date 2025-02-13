-- +goose up
CREATE TABLE ratelimits.ratelimits_identifier_latency_stats_per_hour_v1
(
    time DateTime,
    workspace_id String,
    namespace_id String,
    identifier String,
    request_id String,
    avg_latency AggregateFunction(avg, Int64),
    p99_latency AggregateFunction(quantile(0.99), Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, namespace_id, time, identifier);

-- +goose down
DROP TABLE ratelimits.ratelimits_identifier_latency_stats_per_hour_v1;
