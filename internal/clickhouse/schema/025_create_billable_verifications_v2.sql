-- +goose up
CREATE TABLE billing.billable_verifications_per_month_v2
(
  year          Int,
  month         Int,
  workspace_id  String,
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, year, month)
;

CREATE MATERIALIZED VIEW billing.billable_verifications_per_month_mv_v2
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
-- +goose down

DROP VIEW billing.billable_verifications_per_month_mv_v2;
DROP TABLE billing.billable_verifications_per_month_v2;
