-- name: InsertDomain :exec
INSERT INTO domains (
    id,
    workspace_id,
    project_id,
    domain,
    type,
    subdomain_config,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    CAST(sqlc.arg(subdomain_config) AS JSON),
    ?
) ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    project_id = VALUES(project_id),
    type = VALUES(type),
    subdomain_config = VALUES(subdomain_config),
    updated_at = ?;
