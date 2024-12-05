-- +goose up
ALTER TABLE business.active_workspaces_keys_per_month_mv_v1
MODIFY QUERY
SELECT
  workspace_id, toDate(time) as time
FROM verifications.key_verifications_per_month_v2
;

-- +goose down

ALTER TABLE business.active_workspaces_keys_per_month_mv_v1
MODIFY QUERY
SELECT
  workspace_id, toDate(time) as time
FROM verifications.key_verifications_per_month_v1
;
