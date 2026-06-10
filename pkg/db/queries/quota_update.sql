-- name: UpdateQuota :exec
-- Overwrites a workspace's quota row, team flag included. Unlike UpsertQuota
-- (whose ON DUPLICATE KEY UPDATE deliberately leaves `team` untouched), this
-- sets every field, so it fits full resets like `unkey dev stripe reset`.
UPDATE quota
SET requests_per_month = sqlc.arg(requests_per_month),
    audit_logs_retention_days = sqlc.arg(audit_logs_retention_days),
    logs_retention_days = sqlc.arg(logs_retention_days),
    team = sqlc.arg(team)
WHERE workspace_id = sqlc.arg(workspace_id);
