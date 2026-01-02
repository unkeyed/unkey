-- name: FindSentinelByID :one
SELECT sqlc.embed(s), w.k8s_namespace FROM sentinels s
LEFT JOIN workspaces w ON s.workspace_id = w.id
WHERE s.id = sqlc.arg(id) LIMIT 1;
