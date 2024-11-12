-- +goose up

CREATE TABLE billing.billable_ratelimits_per_month_v1
(
  year          Int,
  month         Int,
  workspace_id  String,
  count         Int64
)
ENGINE = SummingMergeTree()
ORDER BY (workspace_id, year, month)
;

CREATE MATERIALIZED VIEW billing.billable_ratelimits_per_month_mv_v1
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
-- +goose down
DROP VIEW billing.billable_ratelimits_per_month_mv_v1;
DROP TABLE billing.billable_ratelimits_per_month_v1;
