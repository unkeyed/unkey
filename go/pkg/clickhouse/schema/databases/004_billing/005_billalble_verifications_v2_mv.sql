
CREATE MATERIALIZED VIEW IF NOT EXISTS billing.billable_verifications_per_month_mv_v2
TO billing.billable_verifications_per_month_v2
AS SELECT
    workspace_id,
    sum(count) AS count,
    toYear(time) AS year,
    toMonth(time) AS month
FROM verifications.key_verifications_per_month_v1
WHERE outcome = 'VALID'
GROUP BY
    workspace_id,
    year,
    month
;
