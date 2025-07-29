-- name: InsertHostnameRoute :exec
INSERT INTO hostname_routes (
    id,
    workspace_id,
    project_id,
    hostname,
    deployment_id,
    is_enabled,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);