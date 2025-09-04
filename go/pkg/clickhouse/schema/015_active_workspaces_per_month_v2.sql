CREATE TABLE active_workspaces_per_month_v2 (time Date, workspace_id String,) ENGINE = ReplacingMergeTree ()
PARTITION BY
  toYYYYMM (time)
ORDER BY
  (time, workspace_id);

CREATE MATERIALIZED VIEW active_workspaces_keys_per_month_mv_v2 TO active_workspaces_per_month_v2 AS
SELECT
  workspace_id,
  toDate (time) as time
FROM
  key_verifications_per_month_v2;

CREATE MATERIALIZED VIEW active_workspaces_ratelimits_per_month_mv_v2 TO active_workspaces_per_month_v2 AS
SELECT
  workspace_id,
  toDate (time) as time
FROM
  ratelimits_per_month_v2;
