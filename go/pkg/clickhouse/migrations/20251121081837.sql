-- Drop "active_workspaces_keys_per_month_mv_v2" view
DROP VIEW `default`.`active_workspaces_keys_per_month_mv_v2`;
-- Create "active_workspaces_keys_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`active_workspaces_keys_per_month_mv_v2` TO `default`.`active_workspaces_per_month_v2` AS SELECT workspace_id, toDate(time) AS time FROM default.key_verifications_per_month_v3;
-- Drop "billable_verifications_per_month_mv_v2" view
DROP VIEW `default`.`billable_verifications_per_month_mv_v2`;
-- Create "billable_verifications_per_month_mv_v2" view
CREATE MATERIALIZED VIEW `default`.`billable_verifications_per_month_mv_v2` TO `default`.`billable_verifications_per_month_v2` AS SELECT workspace_id, sum(count) AS count, toYear(time) AS year, toMonth(time) AS month FROM default.key_verifications_per_month_v3 WHERE outcome = 'VALID' GROUP BY workspace_id, year, month;
