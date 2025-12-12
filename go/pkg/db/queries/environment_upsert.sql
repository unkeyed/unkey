-- name: UpsertEnvironment :exec
INSERT INTO environments (
    id,
    workspace_id,
    project_id,
    slug,
    sentinel_config,
    created_at
) VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE slug = VALUES(slug);
