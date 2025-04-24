
CREATE MATERIALIZED VIEW IF NOT EXISTS billing.billable_ratelimits_per_month_mv_v1
TO billing.billable_ratelimits_per_month_v1
AS SELECT
    workspace_id,
    sum(passed) AS count,
    toYear(time) AS year,
    toMonth(time) AS month
FROM ratelimits.ratelimits_per_month_v1
WHERE passed > 0
GROUP BY
    workspace_id,
    year,
    month
;
