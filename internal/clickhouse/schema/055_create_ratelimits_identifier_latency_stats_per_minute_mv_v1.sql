-- +goose up
CREATE MATERIALIZED VIEW ratelimits.ratelimits_identifier_latency_stats_per_minute_mv_v1
TO ratelimits.ratelimits_identifier_latency_stats_per_minute_v1
AS
SELECT 
    toStartOfMinute(fromUnixTimestamp64Milli(time)) AS time,
    r.workspace_id,
    r.namespace_id,
    r.identifier,
    r.request_id,
    avgState(m.service_latency) as avg_latency,
    quantileState(0.99)(m.service_latency) as p99_latency
FROM ratelimits.raw_ratelimits_v1 r
JOIN metrics.raw_api_requests_v1 m
ON r.request_id = m.request_id
GROUP BY
    time,
    r.workspace_id,
    r.namespace_id,
    r.identifier,
    r.request_id;

-- +goose down
DROP VIEW ratelimits.ratelimits_identifier_latency_stats_per_minute_mv_v1;
