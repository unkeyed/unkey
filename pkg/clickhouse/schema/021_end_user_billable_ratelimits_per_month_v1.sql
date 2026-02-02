-- End User Billing Billable Rate Limits Per Month Table and Materialized View
-- Created for per-end-user billing feature
-- This table tracks billable rate limits grouped by workspace, external_id, year, and month

CREATE TABLE IF NOT EXISTS default.end_user_billable_ratelimits_per_month_v1 (
    workspace_id String,
    external_id String,
    year Int16,
    month Int8,
    count Int64
) ENGINE = SummingMergeTree()
ORDER BY (workspace_id, external_id, year, month)
TTL toDate(time) + INTERVAL 1 YEAR DELETE;

CREATE MATERIALIZED VIEW IF NOT EXISTS default.end_user_billable_ratelimits_mv_v1 TO default.end_user_billable_ratelimits_per_month_v1 AS
SELECT
    workspace_id,
    identifier AS external_id,
    toYear(time) AS year,
    toMonth(time) AS month,
    sum(passed) AS count
FROM default.ratelimits_per_month_v2
WHERE
    identifier != ''
GROUP BY
    workspace_id,
    identifier,
    year,
    month;