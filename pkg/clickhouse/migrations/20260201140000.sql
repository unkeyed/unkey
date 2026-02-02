-- End User Billing Materialized Views
-- Created for per-end-user billing feature
-- Tables to track billable verifications and credits grouped by workspace, external_id, year, and month

-- End User Billable Verifications Per Month
CREATE TABLE IF NOT EXISTS default.end_user_billable_verifications_per_month_v1 (
    workspace_id String,
    external_id String,
    year Int16,
    month Int8,
    count Int64
) ENGINE = SummingMergeTree()
ORDER BY (workspace_id, external_id, year, month);

CREATE MATERIALIZED VIEW IF NOT EXISTS default.end_user_billable_verifications_mv_v1
TO default.end_user_billable_verifications_per_month_v1 AS
SELECT
    workspace_id,
    external_id,
    toYear(time) AS year,
    toMonth(time) AS month,
    sum(count) AS count
FROM default.key_verifications_per_month_v3
WHERE
    outcome = 'VALID'
    AND external_id != ''
GROUP BY
    workspace_id,
    external_id,
    year,
    month;

-- End User Billable Credits Per Month
CREATE TABLE IF NOT EXISTS default.end_user_billable_credits_per_month_v1 (
    workspace_id String,
    external_id String,
    year Int16,
    month Int8,
    count Int64
) ENGINE = SummingMergeTree()
ORDER BY (workspace_id, external_id, year, month);

-- TODO: Add materialized view for credits tracking when data source is identified
-- The MV should track credit spending similar to verifications but with a filter for credits spent
-- Example query structure (requires identifying the data source):
-- CREATE MATERIALIZED VIEW IF NOT EXISTS default.end_user_billable_credits_mv_v1
-- TO default.end_user_billable_credits_per_month_v1 AS
-- SELECT
--     workspace_id,
--     external_id,
--     toYear(time) AS year,
--     toMonth(time) AS month,
--     sum(count) AS count
-- FROM <source_table>
-- WHERE
--     <credit_spent_condition>
--     AND external_id != ''
-- GROUP BY
--     workspace_id,
--     external_id,
--     year,
--     month;