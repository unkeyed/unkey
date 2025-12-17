-- name: FindSentinelByID :one
SELECT s.*, w.k8s_namespace FROM sentinels s
INNER JOIN `workspaces` w ON s.workspace_id = w.id
WHERE s.id = sqlc.arg(id) LIMIT 1;
