-- name: FindAnalyticsConfigByWorkspaceID :one
SELECT * FROM `analytics_config`
WHERE workspace_id = sqlc.arg(workspace_id);