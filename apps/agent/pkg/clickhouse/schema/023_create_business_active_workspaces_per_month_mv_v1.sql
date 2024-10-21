-- +goose up
CREATE MATERIALIZED VIEW business.active_workspaces_per_month_mv_v1
TO business.active_workspaces_per_month_v1
AS
SELECT
  workspaceId, toDate(time) as time
FROM verifications.key_verifications_per_month_v1
UNION ALL
SELECT 
  workspaceId, toDate(time) as time
FROM ratelimits.ratelimits_per_month_v1
;
