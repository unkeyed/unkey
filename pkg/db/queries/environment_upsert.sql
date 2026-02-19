-- name: UpsertEnvironment :exec
INSERT INTO environments (
    id,
    workspace_id,
    project_id,
    slug,
    created_at
) VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE slug = VALUES(slug);
