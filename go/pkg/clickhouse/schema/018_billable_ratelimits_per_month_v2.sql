CREATE TABLE billable_ratelimits_per_month_v2 (
  year Int16,
  month Int8,
  workspace_id String,
  count Int64
) ENGINE = SummingMergeTree ()
ORDER BY
  (workspace_id, year, month);

CREATE MATERIALIZED VIEW billable_ratelimits_per_month_mv_v2 TO billable_ratelimits_per_month_v2 AS
SELECT
  workspace_id,
  sum(passed) AS count,
  toYear (time) AS year,
  toMonth (time) AS month
FROM
  ratelimits_per_month_v2
GROUP BY
  workspace_id,
  year,
  month;
