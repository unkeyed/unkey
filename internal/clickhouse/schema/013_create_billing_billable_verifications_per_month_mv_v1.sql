-- +goose up
CREATE MATERIALIZED VIEW billing.billable_verifications_per_month_mv_v1
TO billing.billable_verifications_per_month_v1
AS
SELECT
  workspace_id,
  count(*) AS count,
  toYear(time) AS year,
  toMonth(time) AS month
FROM verifications.key_verifications_per_month_v1
WHERE outcome = 'VALID'
GROUP BY
  workspace_id,
  year,
  month
;



-- +goose down
DROP VIEW billing.billable_verifications_per_month_mv_v1;
