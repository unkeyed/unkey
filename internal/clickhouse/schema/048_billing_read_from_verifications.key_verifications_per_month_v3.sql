-- +goose up
ALTER TABLE billing.billable_verifications_per_month_mv_v2
MODIFY QUERY
SELECT
    workspace_id,
    sum(count) AS count,
    toYear(time) AS year,
    toMonth(time) AS month
FROM verifications.key_verifications_per_month_v3
WHERE outcome = 'VALID'
GROUP BY
    workspace_id,
    year,
    month

;


-- +goose down

ALTER TABLE billing.billable_verifications_per_month_mv_v2
MODIFY QUERY
SELECT
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
