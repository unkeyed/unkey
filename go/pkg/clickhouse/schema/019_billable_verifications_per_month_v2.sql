-- Billing Billable Verifications Per Month Table and Materialized View
CREATE TABLE billable_verifications_per_month_v2 (
  year Int16,
  month Int8,
  workspace_id String,
  count Int64
) ENGINE = SummingMergeTree ()
ORDER BY
  (workspace_id, year, month);

CREATE MATERIALIZED VIEW billable_verifications_per_month_mv_v2 TO billable_verifications_per_month_v2 AS
SELECT
  workspace_id,
  sum(count) AS count,
  toYear (time) AS year,
  toMonth (time) AS month
FROM
  key_verifications_per_month_v2
WHERE
  outcome = 'VALID'
GROUP BY
  workspace_id,
  year,
  month;
