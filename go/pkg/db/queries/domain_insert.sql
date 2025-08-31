-- name: InsertDomain :exec
INSERT INTO domains (
    id,
    workspace_id,
    project_id,
    deployment_id,
    domain,
    type,
    created_at
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
) ON DUPLICATE KEY UPDATE
    workspace_id = VALUES(workspace_id),
    project_id = VALUES(project_id),
    deployment_id = VALUES(deployment_id),
    type = VALUES(type),
    updated_at = ?;
