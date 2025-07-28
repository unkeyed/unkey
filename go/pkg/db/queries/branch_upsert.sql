-- name: UpsertBranch :exec
INSERT INTO branches (
    id,
    workspace_id,
    project_id,
    name,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?
) ON DUPLICATE KEY UPDATE
    updated_at = VALUES(updated_at);
