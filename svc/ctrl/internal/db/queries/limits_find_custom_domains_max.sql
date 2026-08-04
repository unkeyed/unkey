-- name: FindCustomDomainsMaxByWorkspaceID :one
SELECT custom_domains_max
FROM `limits`
WHERE workspace_id = sqlc.arg(workspace_id)
LIMIT 1;
