-- name: FindQuotaByWorkspaceID :one
SELECT *
FROM `quota`
WHERE workspace_id = sqlc.arg('workspace_id');
